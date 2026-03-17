// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build sourceavailable && !enterprise

package enterprise

import (
	// Needed to ensure the init() method in the local source-available access control service gets run.
	_ "github.com/mattermost/mattermost/server/v8/enterprise/access_control"
)
