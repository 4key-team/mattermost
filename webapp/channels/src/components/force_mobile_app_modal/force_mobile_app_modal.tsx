// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback, useState} from 'react';
import {FormattedMessage, useIntl} from 'react-intl';

import ExternalLink from 'components/external_link';

import './force_mobile_app_modal.scss';

export const IOS_APP_STORE_LINK = 'https://apps.apple.com/app/mattermost/id1257222717';
export const ANDROID_PLAY_STORE_LINK = 'https://play.google.com/store/apps/details?id=com.mattermost.rn';

const AppleIcon = () => (
    <svg
        className='ForceMobileAppModal__storeIcon'
        viewBox='0 0 24 24'
        width='28'
        height='28'
        aria-hidden={true}
        focusable={false}
    >
        <path
            fill='currentColor'
            d='M16.365 1.43c0 1.14-.417 2.2-1.25 3.06-.9.93-1.99 1.47-3.17 1.38a3.6 3.6 0 0 1 1.27-3.07c.65-.66 1.55-1.16 2.7-1.36.05.32.05.65.05.99zM20.5 17.1c-.34.78-.5 1.13-.94 1.83-.61.97-1.47 2.18-2.54 2.19-.95.01-1.2-.62-2.48-.61-1.29.01-1.56.62-2.51.61-1.07-.01-1.88-1.1-2.49-2.07-1.71-2.71-1.89-5.89-.83-7.58.75-1.2 1.94-1.9 3.06-1.9 1.14 0 1.86.62 2.8.62.91 0 1.47-.63 2.79-.63 1 0 2.06.55 2.81 1.49-2.47 1.35-2.07 4.88.16 5.65z'
        />
    </svg>
);

const GooglePlayIcon = () => (
    <svg
        className='ForceMobileAppModal__storeIcon'
        viewBox='0 0 24 24'
        width='26'
        height='26'
        aria-hidden={true}
        focusable={false}
    >
        <path
            fill='#34A853'
            d='M3.6 2.1c-.3.2-.5.6-.5 1.1v17.6c0 .5.2.9.5 1.1l9.3-9.9L3.6 2.1z'
        />
        <path
            fill='#FBBC04'
            d='M16.9 8.7l-3.9 2.4-2.1 2.2 2.1 2.2 3.9-2.4c1.1-.6 1.1-2.2 0-2.8z'
        />
        <path
            fill='#EA4335'
            d='M3.6 2.1l9.3 10 2.6-2.7L5.6 1.4C4.9 1 4.1 1.7 3.6 2.1z'
        />
        <path
            fill='#4285F4'
            d='M3.6 21.9c.5.4 1.3 1.1 2 .7l9.9-5.7-2.6-2.7-9.3 7.7z'
        />
    </svg>
);

const ForceMobileAppModal = () => {
    const {formatMessage} = useIntl();
    const serverUrl = window.location.origin;
    const [copied, setCopied] = useState(false);

    const handleCopy = useCallback(() => {
        const done = () => {
            setCopied(true);
            window.setTimeout(() => setCopied(false), 2000);
        };
        if (navigator.clipboard?.writeText) {
            navigator.clipboard.writeText(serverUrl).then(done).catch(() => {
                // ignore clipboard errors; nothing else to do
            });
        }
    }, [serverUrl]);

    return (
        <div
            className='ForceMobileAppModal'
            role='dialog'
            aria-modal={true}
        >
            <div className='ForceMobileAppModal__card'>
                <h1 className='ForceMobileAppModal__title'>
                    <FormattedMessage
                        id='force_mobile_app.title'
                        defaultMessage='Access from a browser is restricted'
                    />
                </h1>
                <p className='ForceMobileAppModal__intro'>
                    <FormattedMessage
                        id='force_mobile_app.intro'
                        defaultMessage='To use the chat, please use the Mattermost mobile app. It only takes a couple of minutes:'
                    />
                </p>

                <ol className='ForceMobileAppModal__steps'>
                    <li className='ForceMobileAppModal__step'>
                        <div className='ForceMobileAppModal__stepTitle'>
                            <FormattedMessage
                                id='force_mobile_app.step1.title'
                                defaultMessage='Step 1. Install the Mattermost app'
                            />
                        </div>
                        <div className='ForceMobileAppModal__links'>
                            <ExternalLink
                                className='ForceMobileAppModal__store'
                                href={IOS_APP_STORE_LINK}
                                location='force_mobile_app_modal'
                                aria-label={formatMessage({id: 'force_mobile_app.appStore.aria', defaultMessage: 'Download on the App Store'})}
                            >
                                <AppleIcon/>
                                <span className='ForceMobileAppModal__storeText'>
                                    <span className='ForceMobileAppModal__storeSmall'>
                                        <FormattedMessage
                                            id='force_mobile_app.store.downloadOn'
                                            defaultMessage='Download on the'
                                        />
                                    </span>
                                    {/* eslint-disable-next-line formatjs/no-literal-string-in-jsx */}
                                    <span className='ForceMobileAppModal__storeBig'>{'App Store'}</span>
                                </span>
                            </ExternalLink>
                            <ExternalLink
                                className='ForceMobileAppModal__store'
                                href={ANDROID_PLAY_STORE_LINK}
                                location='force_mobile_app_modal'
                                aria-label={formatMessage({id: 'force_mobile_app.googlePlay.aria', defaultMessage: 'Get it on Google Play'})}
                            >
                                <GooglePlayIcon/>
                                <span className='ForceMobileAppModal__storeText'>
                                    <span className='ForceMobileAppModal__storeSmall'>
                                        <FormattedMessage
                                            id='force_mobile_app.store.getItOn'
                                            defaultMessage='Get it on'
                                        />
                                    </span>
                                    {/* eslint-disable-next-line formatjs/no-literal-string-in-jsx */}
                                    <span className='ForceMobileAppModal__storeBig'>{'Google Play'}</span>
                                </span>
                            </ExternalLink>
                        </div>
                    </li>

                    <li className='ForceMobileAppModal__step'>
                        <div className='ForceMobileAppModal__stepTitle'>
                            <FormattedMessage
                                id='force_mobile_app.step2.title'
                                defaultMessage='Step 2. Open the app and add a server'
                            />
                        </div>
                        <div className='ForceMobileAppModal__stepBody'>
                            <FormattedMessage
                                id='force_mobile_app.step2.serverUrlLabel'
                                defaultMessage='Server address:'
                            />
                            <div className='ForceMobileAppModal__urlRow'>
                                <code className='ForceMobileAppModal__url'>{serverUrl}</code>
                                <button
                                    type='button'
                                    className='ForceMobileAppModal__copy'
                                    onClick={handleCopy}
                                >
                                    {copied ? (
                                        <FormattedMessage
                                            id='force_mobile_app.copied'
                                            defaultMessage='Copied'
                                        />
                                    ) : (
                                        <FormattedMessage
                                            id='force_mobile_app.copy'
                                            defaultMessage='Copy'
                                        />
                                    )}
                                </button>
                            </div>
                            <p className='ForceMobileAppModal__hint'>
                                <FormattedMessage
                                    id='force_mobile_app.step2.serverNameHint'
                                    defaultMessage='The server display name can be anything — for example, “Mattermost”.'
                                />
                            </p>
                        </div>
                    </li>

                    <li className='ForceMobileAppModal__step'>
                        <div className='ForceMobileAppModal__stepTitle'>
                            <FormattedMessage
                                id='force_mobile_app.step3.title'
                                defaultMessage='Step 3. Sign in'
                            />
                        </div>
                        <div className='ForceMobileAppModal__stepBody'>
                            <FormattedMessage
                                id='force_mobile_app.step3.body'
                                defaultMessage='On the next screen, enter your login and password — the same ones you use to sign in here.'
                            />
                        </div>
                    </li>
                </ol>
            </div>
        </div>
    );
};

export default ForceMobileAppModal;
