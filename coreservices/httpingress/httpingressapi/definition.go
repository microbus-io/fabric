/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package httpingressapi

import (
	"time"

	"github.com/microbus-io/fabric/define"
)

// HINT: This file is the single source of truth for the microservice's API. After editing it, run
// cmd/genservice on the microservice's directory (the parent of this api package) to regenerate client.go,
// intermediate.go, mock.go, mock_test.go, and manifest.yaml. Do not hand-edit those generated files.

// Hostname is the default hostname of the microservice.
const Hostname = "http.ingress.core"

// Name is the decorative PascalCase name of the microservice.
const Name = "HTTPIngress"

// Version is a generation counter bumped on each regeneration, not a semantic version.
const Version = 381

// Description is the human-readable summary of the microservice, surfaced in OpenAPI and discovery.
const Description = `The HTTP ingress microservice relays incoming HTTP requests to the NATS bus.`

/*
TimeBudget specifies the timeout for handling a request, after it has been read.
*/
var TimeBudget = define.Config{ // MARKER: TimeBudget
	Value:      time.Duration(0),
	Default:    "20s",
	Validation: "dur [1s,15m]",
}

/*
Ports is a comma-separated list of HTTP ports on which to listen for requests. A port may be
followed by a "tls" marker, e.g. "80, 443 tls, 8080", to terminate TLS using the SAN-indexed
certificates; a bare port enables TLS only when its legacy httpingress-{port}-cert.pem and -key.pem
files are present. Port 80 is always plaintext.
*/
var Ports = define.Config{ // MARKER: Ports
	Value:    string(""),
	Default:  "8080",
	Callback: true,
}

/*
RequestMemoryLimit is the memory capacity used to hold pending requests, in megabytes.
*/
var RequestMemoryLimit = define.Config{ // MARKER: RequestMemoryLimit
	Value:      int(0),
	Default:    "4096",
	Validation: "int [1,]",
}

/*
AllowedOrigins is DEPRECATED. It has been split into AllowedCredentialedOrigins and
AllowedUncredentialedOrigins so that an origin's access to credentials is always explicit.
Setting this config to any non-empty value causes the microservice to refuse to start,
rather than silently ignore an operator's intended posture.

Deprecated: Use AllowedCredentialedOrigins or AllowedUncredentialedOrigins instead
*/
var AllowedOrigins = define.Config{ // MARKER: AllowedOrigins
	Value:    string(""),
	Callback: true,
}

/*
AllowedCredentialedOrigins is a comma-separated list of CORS origins trusted to make credentialed requests.
A listed origin is reflected in Access-Control-Allow-Origin along with Access-Control-Allow-Credentials: true,
permitting the browser to send and read authenticated (cookie-bearing) requests cross-origin.
The wildcard origin is rejected: combining any-origin with credentials is the classic CORS vulnerability,
and this config exists to make it inexpressible.
When both origin lists are empty (the default), Access-Control-Allow-Origin is pinned to the request's
own scheme://host, which permits only same-origin browser reads.
*/
var AllowedCredentialedOrigins = define.Config{ // MARKER: AllowedCredentialedOrigins
	Value:    string(""),
	Callback: true,
}

/*
AllowedUncredentialedOrigins is a comma-separated list of CORS origins allowed to make uncredentialed
requests, or * to allow all origins. The browser blocks credentials for these origins, which is the
correct semantics for a public API. An origin listed in AllowedCredentialedOrigins takes precedence.
When both origin lists are empty (the default), Access-Control-Allow-Origin is pinned to the request's
own scheme://host, which permits only same-origin browser reads.
*/
var AllowedUncredentialedOrigins = define.Config{ // MARKER: AllowedUncredentialedOrigins
	Value:    string(""),
	Callback: true,
}

/*
PortMappings is REMOVED. The x:y->z port-rewrite model has been replaced by AllowedInternalPorts
(internal-port allowlist, no rewrite). Setting this config to any non-empty value causes the
microservice to refuse to start, rather than silently ignore an operator's intended posture.
*/
var PortMappings = define.Config{ // MARKER: PortMappings
	Value:    string(""),
	Callback: true,
}

/*
AllowedInternalPorts is the operator-tunable allowlist of internal destination ports the
ingress is willing to forward to, in addition to the implicitly-allowed :443. Entries are
comma-separated and may be a single port or an inclusive range "N-M", e.g. "1234, 10000-11000".
All entries must satisfy 1024 <= port <= 65535; the microservice refuses to start otherwise.
Ports :666 and :888 are hard-blocked in every deployment mode and cannot be allowlisted. In LOCAL
deployment this config is ignored and every port except :666 and :888 is reachable.
*/
var AllowedInternalPorts = define.Config{ // MARKER: AllowedInternalPorts
	Value:    string(""),
	Callback: true,
}

/*
ReadTimeout specifies the timeout for fully reading a request.
*/
var ReadTimeout = define.Config{ // MARKER: ReadTimeout
	Value:      time.Duration(0),
	Default:    "5m",
	Validation: "dur [1s,]",
	Callback:   true,
}

/*
WriteTimeout specifies the timeout for fully writing the response to a request.
*/
var WriteTimeout = define.Config{ // MARKER: WriteTimeout
	Value:      time.Duration(0),
	Default:    "5m",
	Validation: "dur [1s,]",
	Callback:   true,
}

/*
ReadHeaderTimeout specifies the timeout for fully reading the header of a request.
*/
var ReadHeaderTimeout = define.Config{ // MARKER: ReadHeaderTimeout
	Value:      time.Duration(0),
	Default:    "20s",
	Validation: "dur [1s,]",
	Callback:   true,
}

/*
A newline-separated list of paths or extensions to block with a 404.
Paths should not include any arguments and are matched exactly.
Extensions are specified with "*.ext" and are matched against the extension of the path only.
*/
var BlockedPaths = define.Config{ // MARKER: BlockedPaths
	Value: string(""),
	Default: `/geoserver
/console/
/.env
/.amazon_aws
/solr/admin/info/system
/remote/login
/Autodiscover/Autodiscover.xml
/autodiscover/autodiscover.json
/api/v2/static/not.found
/api/sonicos/tfa
/_ignition/execute-solution
/admin.html
/auth.html
/auth1.html
/readme.txt
/__Additional
/Portal0000.htm
/docs/cplugError.html/
/CSS/Miniweb.css
/.git/*
/cgi-bin/*
/actuator/gateway/routes
/actuator/health
/Public/home/js/check.js
/mifs/.;/services/LogService
/dns-query
/ecp/Current/exporttool/microsoft.exchange.ediscovery.exporttool.application
/owa/auth/x.js
/static/admin/javascript/hetong.js
/sslvpnLogin.html
/vpn/index.html
/wsman
/geoserver/web
/remote/logincheck
/epa/scripts/win/nsepa_setup.exe
/.well-known/security.txt
/cf_scripts/scripts/ajax/ckeditor/ckeditor.js
/Temporary_Listen_Addresses/
/manager/html
/logon/LogonPoint/*
/catalog-portal/ui/oauth/verify
/error_log/.git/HEAD
/containers/json
/hello.world
*.cfm
*.asp
*.aspx
*.cgi
*.jsa
*.jsp
*.shtml
*.php
*.jhtml
*.mwsl
*.dll
*.esp
*.exe`,
	Callback: true,
}
