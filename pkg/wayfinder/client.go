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

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/appvia/wayfinder/pkg/client"
	"github.com/appvia/wayfinder/pkg/client/config"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// configure is reposible for configuring the provider, essentially we do nothing more
// than create a client to the wayfinder api
func configure(ctx context.Context, rd *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var cfg *config.Config

	configfile := fmt.Sprintf("%v", rd.Get("config_file"))
	server := fmt.Sprintf("%v", rd.Get("endpoint"))
	token := fmt.Sprintf("%v", rd.Get("access_token"))

	if server == "" && token == "" && configfile == "" {
		configfile = os.ExpandEnv("$HOME/.wayfinder/config.yaml")
	}

	switch {
	case server != "" && token != "":
		name := "default"
		cfg = &config.Config{CurrentProfile: name}
		cfg.CreateProfile(name, server)
		cfg.AddAuthInfo(name, &config.AuthInfo{Identity: &config.Identity{RefreshToken: token}})
		cfg.AddServer(name, &config.Server{Endpoint: server})

	case configfile != "":
		data, err := os.ReadFile(configfile)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		ccfg, err := config.New(bytes.NewReader(data))
		if err != nil {
			return nil, diag.FromErr(err)
		}
		cfg = ccfg

	default:
		return nil, diag.Errorf("no authenticated configuration for provider")
	}

	// @step: create the client from the configuration
	cc, err := client.New(cfg, client.UseUpdateHandler(updateClientConfiguration(cfg)))
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return cc, diag.Diagnostics{}
}

// updateClientConfiguration is called to update the client configuration
func updateClientConfiguration(cfg *config.Config) client.UpdateHandlerFunc {
	return func() error {
		return nil
	}
}
