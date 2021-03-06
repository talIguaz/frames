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

package kv

import (
	"fmt"
	"strings"

	"github.com/nuclio/logger"
	"github.com/v3io/v3io-go-http"

	"github.com/v3io/frames"
	"github.com/v3io/frames/v3ioutils"
)

// Backend is key/value backend
type Backend struct {
	logger       logger.Logger
	numWorkers   int
	framesConfig *frames.Config
}

// NewBackend return a new key/value backend
func NewBackend(logger logger.Logger, config *frames.BackendConfig, framesConfig *frames.Config) (frames.DataBackend, error) {

	frames.InitBackendDefaults(config, framesConfig)
	newBackend := Backend{
		logger:       logger.GetChild("kv"),
		numWorkers:   config.Workers,
		framesConfig: framesConfig,
	}

	return &newBackend, nil
}

// Create creates a table
func (b *Backend) Create(request *frames.CreateRequest) error {
	return fmt.Errorf("not requiered, table is created on first write")
}

// Delete deletes a table (or part of it)
func (b *Backend) Delete(request *frames.DeleteRequest) error {

	if !strings.HasSuffix(request.Table, "/") {
		request.Table += "/"
	}

	container, err := b.newContainer(request.Session)
	if err != nil {
		return err
	}

	return v3ioutils.DeleteTable(b.logger, container, request.Table, request.Filter, b.numWorkers)
	// TODO: delete the table directory entry if filter == ""
}

// Exec executes a command
func (b *Backend) Exec(request *frames.ExecRequest) error {
	// FIXME
	cmd := strings.TrimSpace(strings.ToLower(request.Command))
	switch cmd {
	case "infer", "inferschema":
		return b.inferSchema(request)
	case "update":
		return b.updateItem(request)
	}
	return fmt.Errorf("KV backend does not support Exec")
}

func (b *Backend) updateItem(request *frames.ExecRequest) error {
	container, err := b.newContainer(request.Session)
	if err != nil {
		return err
	}

	condition := ""
	if val, ok := request.Args["condition"]; ok {
		condition = val.GetSval()
	}

	b.logger.DebugWith("update item", "path", request.Table, "expr", request.Expression)
	return container.Sync.UpdateItem(&v3io.UpdateItemInput{
		Path: request.Table, Expression: &request.Expression, Condition: condition})
}

func (b *Backend) newContainer(session *frames.Session) (*v3io.Container, error) {

	container, err := v3ioutils.NewContainer(
		session,
		b.framesConfig,
		b.logger,
		b.numWorkers,
	)

	return container, err
}
