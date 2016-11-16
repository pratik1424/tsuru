// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dockermachine

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	tsuruErrors "github.com/tsuru/tsuru/errors"
	"github.com/tsuru/tsuru/iaas"
	"github.com/tsuru/tsuru/log"
)

var errDriverNotSet = errors.Errorf("driver is mandatory")

func init() {
	iaas.RegisterIaasProvider("dockermachine", newDockerMachineIaaS)
}

type dockerMachineIaaS struct {
	base       iaas.UserDataIaaS
	apiFactory func(DockerMachineConfig) (DockerMachineAPI, error)
}

func newDockerMachineIaaS(name string) iaas.IaaS {
	return &dockerMachineIaaS{
		base:       iaas.UserDataIaaS{NamedIaaS: iaas.NamedIaaS{BaseIaaSName: "dockermachine", IaaSName: name}},
		apiFactory: NewDockerMachine,
	}
}

func (i *dockerMachineIaaS) getParamOrConfigString(name string, params map[string]string) (string, error) {
	if v, ok := params[name]; ok {
		return v, nil
	}
	return i.base.GetConfigString(name)
}

func (i *dockerMachineIaaS) CreateMachine(params map[string]string) (*iaas.Machine, error) {
	caPath, _ := i.base.GetConfigString("ca-path")
	driverName, ok := params["driver"]
	if !ok {
		name, errConf := i.base.GetConfigString("driver:name")
		if errConf != nil {
			return nil, errDriverNotSet
		}
		driverName = name
		params["driver"] = driverName
	}
	dockerEngineInstallURL, _ := i.getParamOrConfigString("docker-install-url", params)
	insecureRegistry, _ := i.getParamOrConfigString("insecure-registry", params)
	machineName, ok := params["name"]
	if !ok {
		id, err := generateRandomID()
		if err != nil {
			return nil, err
		}
		machineName = fmt.Sprintf("%s-%s", params["pool"], id)
	} else {
		delete(params, "name")
	}
	userDataFileParam, err := i.base.GetConfigString("driver:user-data-file-param")
	if err == nil {
		f, errTemp := ioutil.TempFile("", "")
		if errTemp != nil {
			return nil, errors.Wrap(errTemp, "failed to create userdata file")
		}
		defer os.RemoveAll(f.Name())
		userData, errData := i.base.ReadUserData()
		if errData != nil {
			return nil, errors.WithMessage(errData, "failed to read userdata")
		}
		_, errWrite := f.WriteString(userData)
		if errWrite != nil {
			return nil, errors.Wrap(errWrite, "failed to write local userdata file")
		}
		params[userDataFileParam] = f.Name()
	}
	driverOpts := i.buildDriverOpts(driverName, params)
	if userDataFileParam != "" {
		delete(params, userDataFileParam)
	}
	buf := &bytes.Buffer{}
	dockerMachine, err := i.apiFactory(DockerMachineConfig{
		CaPath:    caPath,
		OutWriter: buf,
		ErrWriter: buf,
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		dockerMachine.Close()
		log.Debug(buf.String())
	}()
	m, err := dockerMachine.CreateMachine(CreateMachineOpts{
		Name:                   machineName,
		DriverName:             driverName,
		Params:                 driverOpts,
		InsecureRegistry:       insecureRegistry,
		DockerEngineInstallURL: dockerEngineInstallURL,
	})
	if err != nil {
		if m != nil {
			errRem := dockerMachine.DeleteMachine(m.Base)
			if errRem != nil {
				err = tsuruErrors.NewMultiError(err, errors.WithMessage(errRem, "failed to remove machine after error"))
			}
		}
		return nil, err
	}
	m.Base.CreationParams = params
	return m.Base, nil
}

func (i *dockerMachineIaaS) buildDriverOpts(driverName string, params map[string]string) map[string]interface{} {
	driverOpts := DefaultParamsForDriver(driverName)
	config, _ := i.base.GetConfig("driver:options")
	if config != nil {
		for k, v := range config.(map[interface{}]interface{}) {
			switch k := k.(type) {
			case string:
				driverOpts[k] = v
			}
		}
	}
	for k, v := range params {
		driverOpts[k] = v
	}
	return driverOpts
}

func (i *dockerMachineIaaS) DeleteMachine(m *iaas.Machine) error {
	buf := &bytes.Buffer{}
	dockerMachine, err := i.apiFactory(DockerMachineConfig{
		OutWriter: buf,
		ErrWriter: buf,
	})
	if err != nil {
		return err
	}
	defer func() {
		dockerMachine.Close()
		log.Debug(buf.String())
	}()
	return dockerMachine.DeleteMachine(m)
}

func generateRandomID() (string, error) {
	id := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, id); err != nil {
		return "", errors.Wrap(err, "failed to generate random id")
	}
	return hex.EncodeToString(id), nil
}

func (i *dockerMachineIaaS) Describe() string {
	return `DockerMachine IaaS required params:
  driver=<driver>                         Driver to be used by docker machine. Can be set on the IaaS configuration.

Optional params:
  name=<name>                             Hostname for the created machine
  docker-install-url=<docker-install-url> Remote script to be used for docker installation. Defaults to: http://get.docker.com. Can be set on the IaaS configuration.
  insecure-registry=<insecure-registry>   Registry to be added as insecure-registry to the docker engine. Can be set on the IaaS configuration.
`
}
