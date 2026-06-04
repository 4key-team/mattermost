// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {screen} from '@testing-library/react';
import React from 'react';

import {renderWithContext} from 'tests/react_testing_utils';

import ForceMobileAppGate from './force_mobile_app_gate';

const baseState = (forceMobileApp: string, roles: string) => ({
    entities: {
        general: {config: {ForceMobileApp: forceMobileApp}},
        users: {
            currentUserId: 'user1',
            profiles: {user1: {id: 'user1', roles}},
        },
    },
});

describe('ForceMobileAppGate', () => {
    test('shows overlay for non-admin when enabled', () => {
        renderWithContext(<ForceMobileAppGate/>, baseState('true', 'system_user'));
        expect(screen.getByRole('dialog')).toBeInTheDocument();
    });

    test('hides overlay when disabled', () => {
        renderWithContext(<ForceMobileAppGate/>, baseState('false', 'system_user'));
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });

    test('hides overlay for system admin even when enabled', () => {
        renderWithContext(<ForceMobileAppGate/>, baseState('true', 'system_user system_admin'));
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
});
