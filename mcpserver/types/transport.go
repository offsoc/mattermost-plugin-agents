// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package types

// TransportMode represents the type of transport being used for MCP communication
type TransportMode string

const (
	// TransportModeStdio allows full functionality including local file access
	TransportModeStdio TransportMode = "stdio"
	// TransportModeHTTP provides network-based access with security restrictions
	TransportModeHTTP TransportMode = "http"
)