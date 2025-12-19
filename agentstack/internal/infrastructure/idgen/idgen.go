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

package idgen

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphaelmansuy/agentstack/internal/domain/evaluation"
)

type UUIDGenerator struct{}

func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

func (g *UUIDGenerator) Generate(prefix string) string {
	if prefix != "" {
		return fmt.Sprintf("%s_%s", prefix, uuid.New().String())
	}
	return uuid.New().String()
}

// Ensure UUIDGenerator implements evaluation.IDGenerator.
var _ evaluation.IDGenerator = (*UUIDGenerator)(nil)
