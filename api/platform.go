// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/tsuru/tsuru/app"
	"github.com/tsuru/tsuru/auth"
	"github.com/tsuru/tsuru/errors"
	"github.com/tsuru/tsuru/io"
	"github.com/tsuru/tsuru/permission"
	"github.com/tsuru/tsuru/provision"
	"github.com/tsuru/tsuru/rec"
)

// title: add platform
// path: /platforms
// method: POST
// consume: multipart/form-data
// produce: application/x-json-stream
// responses:
//   200: Platform created
//   400: Invalid data
//   401: Unauthorized
func platformAdd(w http.ResponseWriter, r *http.Request, t auth.Token) error {
	defer r.Body.Close()
	name := r.FormValue("name")
	file, _, _ := r.FormFile("dockerfile_content")
	if file != nil {
		defer file.Close()
	}
	args := make(map[string]string)
	for key, values := range r.Form {
		args[key] = values[0]
	}
	canCreatePlatform := permission.Check(t, permission.PermPlatformCreate)
	if !canCreatePlatform {
		return permission.ErrUnauthorized
	}
	w.Header().Set("Content-Type", "application/x-json-stream")
	keepAliveWriter := io.NewKeepAliveWriter(w, 30*time.Second, "")
	defer keepAliveWriter.Stop()
	writer := &io.SimpleJsonMessageEncoderWriter{Encoder: json.NewEncoder(keepAliveWriter)}
	err := app.PlatformAdd(provision.PlatformOptions{
		Name:   name,
		Args:   args,
		Input:  file,
		Output: writer,
	})
	if err != nil {
		writer.Encode(io.SimpleJsonMessage{Error: err.Error()})
		writer.Write([]byte("Failed to add platform!\n"))
		return nil
	}
	writer.Write([]byte("Platform successfully added!\n"))
	return nil
}

// title: update platform
// path: /platforms/{name}
// method: PUT
// produce: application/x-json-stream
// responses:
//   200: Platform updated
//   401: Unauthorized
//   404: Not found
func platformUpdate(w http.ResponseWriter, r *http.Request, t auth.Token) error {
	defer r.Body.Close()
	name := r.URL.Query().Get(":name")
	file, _, _ := r.FormFile("dockerfile_content")
	if file != nil {
		defer file.Close()
	}
	args := make(map[string]string)
	for key, values := range r.Form {
		args[key] = values[0]
	}
	canUpdatePlatform := permission.Check(t, permission.PermPlatformUpdate)
	if !canUpdatePlatform {
		return permission.ErrUnauthorized
	}
	w.Header().Set("Content-Type", "application/x-json-stream")
	keepAliveWriter := io.NewKeepAliveWriter(w, 30*time.Second, "")
	defer keepAliveWriter.Stop()
	writer := &io.SimpleJsonMessageEncoderWriter{Encoder: json.NewEncoder(keepAliveWriter)}
	err := app.PlatformUpdate(provision.PlatformOptions{
		Name:   name,
		Args:   args,
		Input:  file,
		Output: writer,
	})
	if err != nil {
		if err == app.ErrPlatformNotFound {
			return &errors.HTTP{Code: http.StatusNotFound, Message: err.Error()}
		}
		writer.Encode(io.SimpleJsonMessage{Error: err.Error()})
		writer.Write([]byte("Failed to update platform!\n"))
		return nil
	}
	writer.Write([]byte("Platform successfully updated!\n"))
	return nil
}

// title: remove platform
// path: /platforms/{name}
// method: DELETE
// responses:
//   200: Platform removed
//   401: Unauthorized
//   404: Not found
func platformRemove(w http.ResponseWriter, r *http.Request, t auth.Token) error {
	canDeletePlatform := permission.Check(t, permission.PermPlatformDelete)
	if !canDeletePlatform {
		return permission.ErrUnauthorized
	}
	name := r.URL.Query().Get(":name")
	err := app.PlatformRemove(name)
	if err == app.ErrPlatformNotFound {
		return &errors.HTTP{Code: http.StatusNotFound, Message: err.Error()}
	}
	return err
}

// title: platform list
// path: /platforms
// method: GET
// produce: application/json
// responses:
//   200: List platforms
//   204: No content
//   401: Unauthorized
func platformList(w http.ResponseWriter, r *http.Request, t auth.Token) error {
	u, err := t.User()
	if err != nil {
		return err
	}
	rec.Log(u.Email, "platform-list")
	canUsePlat := permission.Check(t, permission.PermPlatformUpdate) ||
		permission.Check(t, permission.PermPlatformCreate)
	platforms, err := app.Platforms(!canUsePlat)
	if err != nil {
		return err
	}
	if len(platforms) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(platforms)
}
