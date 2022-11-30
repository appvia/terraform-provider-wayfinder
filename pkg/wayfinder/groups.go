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

package wayfinder

var (
	versions = map[string]string{
		"accountfactoryv2beta1":  "accountfactory.appvia.io/v2beta1",
		"appv2beta1":             "app.appvia.io/v2beta1",
		"cloudaccessv1alpha1":    "cloudaccess.appvia.io/v1alpha1",
		"cloudaccessv2beta1":     "cloudaccess.appvia.io/v2beta1",
		"cloudresourcev2beta1":   "cloudresources.appvia.io/v2beta1",
		"computev1alpha1":        "compute.appvia.io/v1alpha1",
		"computev2beta1":         "compute.appvia.io/v2beta1",
		"configv1alpha1":         "config.appvia.io/v1alpha1",
		"corev1alpha1":           "core.appvia.io/v1alpha1",
		"costsv1alpha1":          "costs.appvia.io/v1alpha1",
		"dnsv2beta1":             "dns.appvia.io/v2beta1",
		"kyvernov1":              "kyverno.io/v1",
		"networkingv1alpha1":     "network.appvia.io/v1alpha1",
		"networkingv2beta1":      "networking.appvia.io/v2beta1",
		"orgv1alpha1":            "org.appvia.io/v1alpha1",
		"orgv2beta1":             "org.appvia.io/v2beta1",
		"packagev1alpha1":        "packages.appvia.io/v1alpha1",
		"packagev2beta1":         "packages.appvia.io/v2beta1",
		"policydeliveryv1alpha1": "policydelivery.appvia.io/v1alpha1",
		"policyv1alpha1":         "policy.appvia.io/v1alpha1",
		"v1":                     "v1",
	}
)

// GetGroupVersion returns the api version
func GetGroupVersion(short string) string {
	return versions[short]
}
