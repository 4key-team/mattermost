// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {useSelector} from 'react-redux';

import {getConfig} from 'mattermost-redux/selectors/entities/general';
import {isCurrentUserSystemAdmin} from 'mattermost-redux/selectors/entities/users';

import ForceMobileAppModal from './force_mobile_app_modal';

const ForceMobileAppGate = () => {
    const config = useSelector(getConfig);
    const isAdmin = useSelector(isCurrentUserSystemAdmin);

    const enabled = config?.ForceMobileApp === 'true';

    if (!enabled || isAdmin) {
        return null;
    }

    return <ForceMobileAppModal/>;
};

export default ForceMobileAppGate;
