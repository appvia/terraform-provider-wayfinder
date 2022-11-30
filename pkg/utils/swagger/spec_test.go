/*
 * Copyright (C) 2022  Appvia Ltd <info@appvia.io>
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License
 * as published by the Free Software Foundation; either version 2
 * of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package swagger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasSwaggerDefinitionBad(t *testing.T) {
	assert.False(t, HasDefinition("test"))
	assert.False(t, HasDefinition("#definitions"))
}

func TestHasSwaggerDefinitionOk(t *testing.T) {
	assert.True(t, HasDefinition("#/definitions/"))
}

func TestGetResourceKindFromReference(t *testing.T) {
	assert.Equal(t, "Test", GetResourceKindFromReference("#/definitions/corev1alpha1.Test"))
	assert.Equal(t, "Test", GetResourceKindFromReference("Test"))
	assert.Equal(t, "Test", GetResourceKindFromReference("corev1alpha1.Test"))
}

func TestGetResourceAPIGroupFromReference(t *testing.T) {
	assert.Equal(t, "corev1alpha1", GetResourceAPIGroupFromReference("#/definitions/corev1alpha1.Test"))
	assert.Equal(t, "corev1alpha1", GetResourceAPIGroupFromReference("corev1alpha1.Test"))
	assert.Equal(t, "corev1alpha1", GetResourceAPIGroupFromReference("corev1alpha1"))
}
