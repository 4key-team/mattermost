// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback, useEffect, useRef, useState} from 'react';
import {useIntl} from 'react-intl';

import {GenericModal} from '@mattermost/components';
import type {UserProfile} from '@mattermost/types/users';

import type {PasswordConfig} from 'mattermost-redux/selectors/entities/general';
import type {ActionResult} from 'mattermost-redux/types/actions';
import {isEmail} from 'mattermost-redux/utils/helpers';

import Input from 'components/widgets/inputs/input/input';

import {isValidPassword} from 'utils/password';
import {copyToClipboard} from 'utils/utils';

import '../admin_modal_with_input.scss';

type Props = {
    onExited: () => void;
    onSuccess: (user: UserProfile) => void;
    passwordConfig: PasswordConfig;
    actions: {
        createUser: (user: UserProfile, token: string, inviteId: string, redirect: string) => Promise<ActionResult<UserProfile>>;
    };
}

export default function CreateUserModal({
    onExited,
    onSuccess,
    passwordConfig,
    actions,
}: Props) {
    const {formatMessage} = useIntl();

    const [show, setShow] = useState(true);
    const [email, setEmail] = useState('');
    const [username, setUsername] = useState('');
    const [firstName, setFirstName] = useState('');
    const [lastName, setLastName] = useState('');
    const [position, setPosition] = useState('');
    const [password, setPassword] = useState('');
    const [copied, setCopied] = useState(false);
    const [serverError, setServerError] = useState<React.ReactNode>(null);
    const [emailError, setEmailError] = useState<React.ReactNode>(null);
    const [usernameError, setUsernameError] = useState<React.ReactNode>(null);
    const [passwordError, setPasswordError] = useState<React.ReactNode>(null);

    const emailRef = useRef<HTMLInputElement>(null);
    const usernameRef = useRef<HTMLInputElement>(null);
    const passwordRef = useRef<HTMLInputElement>(null);
    const copyTimeoutRef = useRef<number | null>(null);

    useEffect(() => {
        return () => {
            if (copyTimeoutRef.current) {
                window.clearTimeout(copyTimeoutRef.current);
            }
        };
    }, []);

    const handleGeneratePassword = useCallback(() => {
        const lowercase = 'abcdefghijklmnopqrstuvwxyz';
        const uppercase = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';
        const numbers = '0123456789';
        const symbols = '!@#$%^&*()-_=+[]{};:,.?';

        const requiredSets = [];
        if (passwordConfig.requireLowercase) {
            requiredSets.push(lowercase);
        }
        if (passwordConfig.requireUppercase) {
            requiredSets.push(uppercase);
        }
        if (passwordConfig.requireNumber) {
            requiredSets.push(numbers);
        }
        if (passwordConfig.requireSymbol) {
            requiredSets.push(symbols);
        }
        if (requiredSets.length === 0) {
            requiredSets.push(lowercase, uppercase, numbers);
        }

        const allChars = `${lowercase}${uppercase}${numbers}${symbols}`;
        const targetLength = Math.max(passwordConfig.minimumLength || 10, requiredSets.length, 16);

        const randomInt = (max: number) => {
            if (window.crypto?.getRandomValues) {
                const array = new Uint32Array(1);
                window.crypto.getRandomValues(array);
                return array[0] % max;
            }
            return Math.floor(Math.random() * max);
        };

        const pickChar = (source: string) => source[randomInt(source.length)];

        const generatedChars = requiredSets.map((set) => pickChar(set));
        while (generatedChars.length < targetLength) {
            generatedChars.push(pickChar(allChars));
        }

        for (let i = generatedChars.length - 1; i > 0; i--) {
            const swapIndex = randomInt(i + 1);
            [generatedChars[i], generatedChars[swapIndex]] = [generatedChars[swapIndex], generatedChars[i]];
        }

        const generatedPassword = generatedChars.join('');
        setPassword(generatedPassword);
        setPasswordError(null);
        setServerError(null);
    }, [passwordConfig]);

    const handleCopyPassword = useCallback(() => {
        if (!password) {
            return;
        }

        copyToClipboard(password);
        setCopied(true);

        if (copyTimeoutRef.current) {
            window.clearTimeout(copyTimeoutRef.current);
        }

        copyTimeoutRef.current = window.setTimeout(() => {
            setCopied(false);
            copyTimeoutRef.current = null;
        }, 5000);
    }, [password]);

    const handleCancel = useCallback(() => {
        setShow(false);
    }, []);

    const handleConfirm = useCallback(async () => {
        setServerError(null);
        setEmailError(null);
        setUsernameError(null);
        setPasswordError(null);

        const normalizedEmail = email.trim().toLowerCase();
        const normalizedUsername = username.trim();

        if (!normalizedEmail || !isEmail(normalizedEmail)) {
            setEmailError(formatMessage({
                id: 'user.settings.general.validEmail',
                defaultMessage: 'Please enter a valid email address.',
            }));
            return;
        }

        if (!normalizedUsername) {
            setUsernameError(formatMessage({
                id: 'admin.system_users.create_user.username_required',
                defaultMessage: 'Please enter a username.',
            }));
            return;
        }

        const {valid, error} = isValidPassword(password, passwordConfig);
        if (!valid && error) {
            setPasswordError(error);
            return;
        }

        const userToCreate = {
            email: normalizedEmail,
            username: normalizedUsername,
            first_name: firstName.trim(),
            last_name: lastName.trim(),
            position: position.trim(),
            password,
        } as UserProfile;

        const result = await actions.createUser(userToCreate, '', '', '');
        if ('error' in result) {
            const resultError = result.error;
            if (resultError.server_error_id === 'model.user.is_valid.email.app_error' || resultError.server_error_id === 'app.user.save.email_exists.app_error') {
                setEmailError(resultError.message);
            } else if (resultError.server_error_id === 'model.user.is_valid.username.app_error' || resultError.server_error_id === 'app.user.save.username_exists.app_error') {
                setUsernameError(resultError.message);
            } else if (resultError.server_error_id === 'model.user.is_valid.pwd.app_error') {
                setPasswordError(resultError.message);
            } else {
                setServerError(resultError.message);
            }
            return;
        }

        onSuccess(result.data);
        setShow(false);
    }, [actions, email, firstName, formatMessage, lastName, onSuccess, password, passwordConfig, position, username]);

    return (
        <GenericModal
            id='createUserModal'
            className='CreateUserModal'
            modalHeaderText={formatMessage({
                id: 'admin.system_users.create_user.title',
                defaultMessage: 'Create user',
            })}
            show={show}
            onExited={onExited}
            onHide={handleCancel}
            handleCancel={handleCancel}
            handleConfirm={handleConfirm}
            handleEnterKeyPress={handleConfirm}
            confirmButtonText={formatMessage({
                id: 'admin.system_users.create_user.confirm',
                defaultMessage: 'Create',
            })}
            compassDesign={true}
            autoCloseOnConfirmButton={false}
            errorText={serverError ? <span className='error'>{serverError}</span> : undefined}
        >
            <div className='CreateUserModal__body'>
                <Input
                    ref={emailRef as React.Ref<HTMLInputElement>}
                    type='email'
                    name='email'
                    autoComplete='off'
                    label={formatMessage({
                        id: 'admin.system_users.create_user.email',
                        defaultMessage: 'Email',
                    })}
                    placeholder={formatMessage({
                        id: 'admin.system_users.create_user.email_placeholder',
                        defaultMessage: 'Enter email address',
                    })}
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    autoFocus={true}
                    maxLength={128}
                    customMessage={emailError ? {type: 'error', value: emailError} : undefined}
                />
                <Input
                    ref={usernameRef as React.Ref<HTMLInputElement>}
                    type='text'
                    name='username'
                    autoComplete='off'
                    label={formatMessage({
                        id: 'admin.system_users.create_user.username',
                        defaultMessage: 'Username',
                    })}
                    placeholder={formatMessage({
                        id: 'admin.system_users.create_user.username_placeholder',
                        defaultMessage: 'Enter username',
                    })}
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    maxLength={64}
                    customMessage={usernameError ? {type: 'error', value: usernameError} : undefined}
                />
                <Input
                    type='text'
                    name='first_name'
                    autoComplete='off'
                    label={formatMessage({
                        id: 'admin.system_users.create_user.first_name',
                        defaultMessage: 'First name',
                    })}
                    placeholder={formatMessage({
                        id: 'admin.system_users.create_user.first_name_placeholder',
                        defaultMessage: 'Enter first name',
                    })}
                    value={firstName}
                    onChange={(e) => setFirstName(e.target.value)}
                    maxLength={64}
                />
                <Input
                    type='text'
                    name='last_name'
                    autoComplete='off'
                    label={formatMessage({
                        id: 'admin.system_users.create_user.last_name',
                        defaultMessage: 'Last name',
                    })}
                    placeholder={formatMessage({
                        id: 'admin.system_users.create_user.last_name_placeholder',
                        defaultMessage: 'Enter last name',
                    })}
                    value={lastName}
                    onChange={(e) => setLastName(e.target.value)}
                    maxLength={64}
                />
                <Input
                    type='text'
                    name='position'
                    autoComplete='off'
                    label={formatMessage({
                        id: 'admin.system_users.create_user.position',
                        defaultMessage: 'Job title',
                    })}
                    placeholder={formatMessage({
                        id: 'admin.system_users.create_user.position_placeholder',
                        defaultMessage: 'Enter job title',
                    })}
                    value={position}
                    onChange={(e) => setPosition(e.target.value)}
                    maxLength={128}
                />
                <Input
                    ref={passwordRef as React.Ref<HTMLInputElement>}
                    type='password'
                    name='password'
                    autoComplete='new-password'
                    label={formatMessage({
                        id: 'admin.system_users.create_user.password',
                        defaultMessage: 'Password',
                    })}
                    placeholder={formatMessage({
                        id: 'admin.system_users.create_user.password_placeholder',
                        defaultMessage: 'Enter password',
                    })}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    customMessage={passwordError ? {type: 'error', value: passwordError} : undefined}
                    inputSuffix={
                        <div className='systemUsers__passwordActions'>
                            <button
                                type='button'
                                className='systemUsers__passwordAction'
                                onClick={handleGeneratePassword}
                            >
                                {formatMessage({
                                    id: 'admin.system_users.create_user.password_generate',
                                    defaultMessage: 'Generate',
                                })}
                            </button>
                            <button
                                type='button'
                                className='systemUsers__passwordAction'
                                onClick={handleCopyPassword}
                                disabled={!password}
                            >
                                {copied ? formatMessage({
                                    id: 'admin.system_users.create_user.password_copied',
                                    defaultMessage: 'Copied',
                                }) : formatMessage({
                                    id: 'admin.system_users.create_user.password_copy',
                                    defaultMessage: 'Copy',
                                })}
                            </button>
                        </div>
                    }
                />
            </div>
        </GenericModal>
    );
}
