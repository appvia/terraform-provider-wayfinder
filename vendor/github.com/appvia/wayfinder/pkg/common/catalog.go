/**
 * Copyright 2021 Appvia Ltd <info@appvia.io>
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

package common

import "fmt"

// WayfinderChartsServiceName is the kubernetes service name which is used to
// expose the helm catalog
const WayfinderChartsServiceName = "wayfinder-charts"

// GetCatalogURL returns the helm catalog endpoint
func GetCatalogURL() string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local",
		WayfinderChartsServiceName, ManagedClusterWFNamespace)
}

// GetHelmRespositoryName returns a helm repository name from the url
func GetHelmRespositoryName(url string) (string, bool) {
	switch url {
	case GetCatalogURL():
		return "wayfinder", true

	case "https://charts.helm.sh/stable":
		return "stable", true

	case "https://charts.helm.sh/incubator":
		return "incubator", true

	case "https://charts.bitnami.com/bitnami":
		return "bitnami", true

	case "https://istio-release.storage.googleapis.com/charts":
		return "istio", true
	}

	return url, false
}
