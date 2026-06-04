// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {FormattedMessage} from 'react-intl';

import './force_mobile_app_modal.scss';

export const IOS_APP_STORE_LINK = 'https://apps.apple.com/app/mattermost/id1257222717';
export const ANDROID_PLAY_STORE_LINK = 'https://play.google.com/store/apps/details?id=com.mattermost.rn';

const ForceMobileAppModal = () => {
    const serverUrl = window.location.origin;

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
                <p className='ForceMobileAppModal__text'>
                    <FormattedMessage
                        id='force_mobile_app.instructions'
                        defaultMessage='To continue, please download the Mattermost mobile app and sign in using this server address:'
                    />
                </p>
                <p className='ForceMobileAppModal__server'>{serverUrl}</p>
                <div className='ForceMobileAppModal__links'>
                    <a
                        className='ForceMobileAppModal__store'
                        href={IOS_APP_STORE_LINK}
                        target='_blank'
                        rel='noopener noreferrer'
                    >
                        <FormattedMessage
                            id='force_mobile_app.appStore'
                            defaultMessage='Download on the App Store'
                        />
                    </a>
                    <a
                        className='ForceMobileAppModal__store'
                        href={ANDROID_PLAY_STORE_LINK}
                        target='_blank'
                        rel='noopener noreferrer'
                    >
                        <FormattedMessage
                            id='force_mobile_app.googlePlay'
                            defaultMessage='Get it on Google Play'
                        />
                    </a>
                </div>
            </div>
        </div>
    );
};

export default ForceMobileAppModal;
