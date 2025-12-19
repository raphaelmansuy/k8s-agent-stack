/*
 * Copyright 2025 Raphaël MANSUY
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package logger

import (
	"testing"
)

func TestNew(t *testing.T) {
	log, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = log.Sync() }()

	if log == nil {
		t.Fatal("New() returned nil logger")
	}
}

func TestNewNop(t *testing.T) {
	log := NewNop()
	if log == nil {
		t.Fatal("NewNop() returned nil logger")
	}
}
