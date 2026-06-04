// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {screen} from '@testing-library/react';
import React from 'react';

import {renderWithContext} from 'tests/react_testing_utils';

import ForceMobileAppModal, {IOS_APP_STORE_LINK, ANDROID_PLAY_STORE_LINK} from './force_mobile_app_modal';

describe('ForceMobileAppModal', () => {
    test('renders both store links and no close control', () => {
        renderWithContext(<ForceMobileAppModal/>);

        const ios = screen.getByRole('link', {name: /app store/i});
        const android = screen.getByRole('link', {name: /google play/i});

        expect(ios).toHaveAttribute('href', IOS_APP_STORE_LINK);
        expect(android).toHaveAttribute('href', ANDROID_PLAY_STORE_LINK);
        expect(screen.queryByRole('button', {name: /close/i})).not.toBeInTheDocument();
    });
});
