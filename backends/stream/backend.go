/*
Copyright 2018 Iguazio Systems Ltd.

Licensed under the Apache License, Version 2.0 (the "License") with
an addition restriction as set forth herein. You may not use this
file except in compliance with the License. You may obtain a copy of
the License at http://www.apache.org/licenses/LICENSE-2.0.

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
implied. See the License for the specific language governing
permissions and limitations under the License.

In addition, you may not use the software for any purposes that are
illegal under applicable law, and the grant of the foregoing license
under the Apache 2.0 license is conditioned upon your compliance with
such restriction.
*/

package stream

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/nuclio/logger"
	"github.com/pkg/errors"

	"github.com/v3io/frames"
	"github.com/v3io/frames/backends"
	"github.com/v3io/frames/v3ioutils"
	v3io "github.com/v3io/v3io-go-http"
)

// Backend is a tsdb backend
type Backend struct {
	backendConfig *frames.BackendConfig
	framesConfig  *frames.Config
	logger        logger.Logger
}

// NewBackend return a new v3io stream backend
func NewBackend(logger logger.Logger, cfg *frames.BackendConfig, framesConfig *frames.Config) (frames.DataBackend, error) {

	frames.InitBackendDefaults(cfg, framesConfig)
	newBackend := Backend{
		logger:        logger.GetChild("stream"),
		backendConfig: cfg,
		framesConfig:  framesConfig,
	}

	return &newBackend, nil
}

// Create creates a table
func (b *Backend) Create(request *frames.CreateRequest) error {

	// TODO: check if Stream exist, if it already has the desired params can silently ignore, may need a -silent flag

	var isInt bool
	attrs := request.Attributes()
	shards := int64(1)

	shardsVar, ok := attrs["shards"]
	if ok {
		shards, isInt = shardsVar.(int64)
		fmt.Println(reflect.TypeOf(shardsVar))
		if !isInt || shards < 1 {
			return errors.Errorf("Shards attribute must be a positive integer (got %v)", shardsVar)
		}
	}

	retention := int64(24)
	retentionVar, ok := attrs["retention_hours"]
	if ok {
		retention, isInt = retentionVar.(int64)
		if !isInt || retention < 1 {
			return errors.Errorf("retention_hours attribute must be a positive integer (got %v)", retentionVar)
		}
	}

	if !strings.HasSuffix(request.Table, "/") {
		request.Table += "/"
	}

	container, err := b.newContainer(request.Session)
	if err != nil {
		return err
	}

	err = container.Sync.CreateStream(&v3io.CreateStreamInput{
		Path: request.Table, ShardCount: int(shards), RetentionPeriodHours: int(retention)})
	if err != nil {
		b.logger.ErrorWith("CreateStream failed", "path", request.Table, "err", err)
	}

	return nil
}

// Delete deletes a table or part of it
func (b *Backend) Delete(request *frames.DeleteRequest) error {

	if !strings.HasSuffix(request.Table, "/") {
		request.Table += "/"
	}

	container, err := b.newContainer(request.Session)
	if err != nil {
		return err
	}

	err = container.Sync.DeleteStream(&v3io.DeleteStreamInput{Path: request.Table})
	if err != nil {
		b.logger.ErrorWith("DeleteStream failed", "path", request.Table, "err", err)
	}

	return nil
}

// Exec executes a command
func (b *Backend) Exec(request *frames.ExecRequest) error {
	// FIXME
	return fmt.Errorf("KV backend does not support Exec")
}

func (b *Backend) newContainer(session *frames.Session) (*v3io.Container, error) {

	container, err := v3ioutils.NewContainer(
		session,
		b.framesConfig,
		b.logger,
		b.backendConfig.Workers,
	)

	return container, err
}

func init() {
	if err := backends.Register("stream", NewBackend); err != nil {
		panic(err)
	}
}
