// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import type {UserProfile} from '@mattermost/types/users';

import {renderWithContext, screen, userEvent, waitFor} from 'tests/react_testing_utils';
import * as Utils from 'utils/utils';

import CreateUserModal from './create_user_modal';

describe('components/admin_console/system_users/create_user_modal', () => {
    const baseProps = {
        actions: {createUser: jest.fn()},
        onExited: jest.fn(),
        onSuccess: jest.fn(),
        passwordConfig: {
            minimumLength: 10,
            requireLowercase: true,
            requireNumber: true,
            requireSymbol: true,
            requireUppercase: true,
        },
    };

    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('should validate required inputs before submitting', async () => {
        renderWithContext(<CreateUserModal {...baseProps}/>);

        await userEvent.click(screen.getByRole('button', {name: /Create/i}));

        await waitFor(() => {
            expect(screen.getByText(/Please enter a valid email address/i)).toBeInTheDocument();
        });

        expect(baseProps.actions.createUser).not.toHaveBeenCalled();
    });

    test('should create user and call onSuccess', async () => {
        const createdUser = {id: 'new_user_id'} as UserProfile;
        const createUser = jest.fn().mockResolvedValue({data: createdUser});
        const onSuccess = jest.fn();

        renderWithContext(
            <CreateUserModal
                {...baseProps}
                actions={{createUser}}
                onSuccess={onSuccess}
            />,
        );

        await userEvent.type(screen.getByLabelText(/Email/i), 'new.user@example.com');
        await userEvent.type(screen.getByLabelText(/Username/i), 'newuser');
        await userEvent.type(screen.getByLabelText(/Password/i), 'Password123!');
        await userEvent.click(screen.getByRole('button', {name: /^Create$/i}));

        await waitFor(() => {
            expect(createUser).toHaveBeenCalledWith(
                {
                    email: 'new.user@example.com',
                    username: 'newuser',
                    password: 'Password123!',
                },
                '',
                '',
                '',
            );
        });
        expect(onSuccess).toHaveBeenCalledWith(createdUser);
    });

    test('should generate a valid password and copy it', async () => {
        const copySpy = jest.spyOn(Utils, 'copyToClipboard').mockImplementation(jest.fn());

        renderWithContext(<CreateUserModal {...baseProps}/>);

        await userEvent.click(screen.getByRole('button', {name: /Generate/i}));

        const passwordInput = screen.getByLabelText(/Password/i) as HTMLInputElement;
        expect(passwordInput.value).toHaveLength(16);

        await userEvent.click(screen.getByRole('button', {name: /Copy/i}));

        expect(copySpy).toHaveBeenCalledWith(passwordInput.value);
    });
});
