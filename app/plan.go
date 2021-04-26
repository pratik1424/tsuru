// Copyright 2014 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package app

import (
	"context"

	"github.com/pkg/errors"
	tsuruErrors "github.com/tsuru/tsuru/errors"
	"github.com/tsuru/tsuru/storage"
	appTypes "github.com/tsuru/tsuru/types/app"
)

var defaultPlans = []appTypes.Plan{
	// general plans
	{
		Name:     "c0.1m0.1",
		CPUMilli: 100,
		Memory:   128 * 1024 * 1024,
	},
	{
		Name:     "c0.1m0.2",
		CPUMilli: 100,
		Memory:   256 * 1024 * 1024,
		Default:  true,
	},
	{
		Name:     "c0.3m0.5",
		CPUMilli: 300,
		Memory:   512 * 1024 * 1024,
	},
	{
		Name:     "c0.5m1",
		CPUMilli: 500,
		Memory:   1024 * 1024 * 1024,
	},
	{
		Name:     "c1m2",
		CPUMilli: 1000,
		Memory:   2 * 1024 * 1024 * 1024,
	},
	{
		Name:     "c2m4",
		CPUMilli: 2000,
		Memory:   4 * 1024 * 1024 * 1024,
	},
	{
		Name:     "c4m8",
		CPUMilli: 4000,
		Memory:   8 * 1024 * 1024 * 1024,
	},
	{
		Name:     "c8m16",
		CPUMilli: 8000,
		Memory:   16 * 1024 * 1024 * 1024,
	},

	// high cpu plans
	{
		Name:     "c1m1",
		CPUMilli: 1000,
		Memory:   1 * 1024 * 1024 * 1024,
	},
	{
		Name:     "c2m2",
		CPUMilli: 2000,
		Memory:   2 * 1024 * 1024 * 1024,
	},
	{
		Name:     "c4m4",
		CPUMilli: 4000,
		Memory:   4 * 1024 * 1024 * 1024,
	},
	{
		Name:     "c8m8",
		CPUMilli: 8000,
		Memory:   8 * 1024 * 1024 * 1024,
	},
	{
		Name:     "c16m16",
		CPUMilli: 16000,
		Memory:   16 * 1024 * 1024 * 1024,
	},

	// extreme cpu plans
	{
		Name:     "c2m1",
		CPUMilli: 2000,
		Memory:   1 * 1024 * 1024 * 1024,
	},
	{
		Name:     "c4m2",
		CPUMilli: 4000,
		Memory:   2 * 1024 * 1024 * 1024,
	},
	{
		Name:     "c8m4",
		CPUMilli: 8000,
		Memory:   4 * 1024 * 1024 * 1024,
	},
	{
		Name:     "c16m8",
		CPUMilli: 16000,
		Memory:   8 * 1024 * 1024 * 1024,
	},

	// high mem plans
	{
		Name:     "c0.3m1",
		CPUMilli: 300,
		Memory:   1024 * 1024 * 1024,
	},
	{
		Name:     "c0.5m2",
		CPUMilli: 500,
		Memory:   2 * 1024 * 1024 * 1024,
	},
	{
		Name:     "c1m4",
		CPUMilli: 1000,
		Memory:   4 * 1024 * 1024 * 1024,
	},
	{
		Name:     "c2m8",
		CPUMilli: 2000,
		Memory:   8 * 1024 * 1024 * 1024,
	},
	{
		Name:     "c4m16",
		CPUMilli: 4000,
		Memory:   16 * 1024 * 1024 * 1024,
	},
}

type planService struct {
	storage appTypes.PlanStorage
}

func PlanService() (appTypes.PlanService, error) {
	dbDriver, err := storage.GetCurrentDbDriver()
	if err != nil {
		dbDriver, err = storage.GetDefaultDbDriver()
		if err != nil {
			return nil, err
		}
	}
	svc := &planService{
		storage: dbDriver.PlanStorage,
	}
	err = svc.ensureDefault(context.Background())
	if err != nil {
		return nil, err
	}
	return svc, nil
}

// Create implements Create method of PlanService interface
func (s *planService) Create(ctx context.Context, plan appTypes.Plan) error {
	if plan.Name == "" {
		return appTypes.PlanValidationError{Field: "name"}
	}
	if plan.CpuShare > 0 && plan.CpuShare < 2 {
		return appTypes.ErrLimitOfCpuShare
	}
	if plan.Memory > 0 && plan.Memory < 4194304 {
		return appTypes.ErrLimitOfMemory
	}
	return s.storage.Insert(ctx, plan)
}

// List implements List method of PlanService interface
func (s *planService) List(ctx context.Context) ([]appTypes.Plan, error) {
	return s.storage.FindAll(ctx)
}

func (s *planService) FindByName(ctx context.Context, name string) (*appTypes.Plan, error) {
	return s.storage.FindByName(ctx, name)
}

// DefaultPlan implements DefaultPlan method of PlanService interface
func (s *planService) DefaultPlan(ctx context.Context) (*appTypes.Plan, error) {
	return s.storage.FindDefault(ctx)
}

// Remove implements Remove method of PlanService interface
func (s *planService) Remove(ctx context.Context, planName string) error {
	return s.storage.Delete(ctx, appTypes.Plan{Name: planName})
}

// ensureDefault creates and stores an autogenerated plan in case of no plans
// exists.
func (s *planService) ensureDefault(ctx context.Context) error {
	plans, err := s.storage.FindAll(ctx)
	if err != nil {
		return err
	}
	if len(plans) > 0 {
		return nil
	}

	multiErr := tsuruErrors.NewMultiError()
	for _, defaultPlan := range defaultPlans {
		err = s.storage.Insert(ctx, defaultPlan)
		if err != nil {
			err = errors.Wrapf(err, "could not insert plan %q", defaultPlan.Name)
			multiErr.Add(err)
		}
	}

	return multiErr.ToError()
}
