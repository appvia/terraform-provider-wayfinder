/*
 * Copyright (C) 2022 Appvia Ltd <info@appvia.io>
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

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/spf13/cobra"

	"github.com/appvia/terraform-provider-wayfinder/pkg/assets"
	"github.com/appvia/terraform-provider-wayfinder/pkg/version"
	"github.com/appvia/terraform-provider-wayfinder/pkg/wayfinder"
)

var (
	// includedResources is a collection of resources to include
	includedResources []string
)

func main() {
	cmd := &cobra.Command{
		Use:     "terraform-provider-wayfinder [options]",
		Short:   "Terraform provider used to configure a Wayfinder instance",
		Version: version.GetVersion(),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := wayfinder.New(wayfinder.Config{
				IncludeResources: includedResources,
				SwaggerSpec:      bytes.NewReader(assets.MustAsset("swagger.json")),
				SkippedResources: []string{
					"wayfinder_costs",
					"wayfinder_idpconfig",
					"workspace_nodepools",
				},
			})
			if err != nil {
				return err
			}

			plugin.Serve(&plugin.ServeOpts{ProviderFunc: svc})

			return nil
		},
	}
	flags := cmd.Flags()
	flags.StringSliceVar(&includedResources, "include-resources", []string{}, "resources to includes")

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "[Error] %s\n", err)

		os.Exit(1)
	}
}
