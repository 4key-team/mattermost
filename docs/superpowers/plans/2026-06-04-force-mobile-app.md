# Force Mobile App (Web Block) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a System Console toggle `ServiceSettings.ForceMobileApp` that, when enabled, shows non-admin web users a full-screen non-dismissable overlay instructing them to use the mobile app, blocking the web client.

**Architecture:** A new server boolean config field is serialized into the client config for all logged-in users. A presentational overlay component renders store links and instructions; a container gate reads `getConfig` + `isCurrentUserSystemAdmin` from Redux and decides whether to render it; the gate is mounted in the authenticated `LoggedIn` layout. The mobile native app is unaffected because it never runs this web code, so no platform detection is needed.

**Tech Stack:** Go (server config), TypeScript/React/Redux (webapp), react-intl (i18n), jest + react-testing-library (`renderWithContext`).

---

## File Structure

**Server (Go):**
- Modify `server/public/model/config.go` — add `ForceMobileApp *bool` field to `ServiceSettings` + default in `SetDefaults`.
- Modify `server/config/client.go` — serialize `ForceMobileApp` into `GenerateClientConfig`.
- Test `server/public/model/config_test.go` — default value test.
- Test `server/config/client_test.go` — presence in client config.

**Webapp types:**
- Modify `webapp/platform/types/src/config.ts` — add `ForceMobileApp` to `ClientConfig` (string) and `AdminConfig` ServiceSettings (boolean).

**Webapp UI:**
- Create `webapp/channels/src/components/force_mobile_app_modal/index.ts` — re-export.
- Create `webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_modal.tsx` — presentational overlay (store links, text). Holds the hardcoded store-link constants.
- Create `webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_modal.scss` — full-screen overlay styles.
- Create `webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_gate.tsx` — container reading Redux, decides whether to render the overlay.
- Test `webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_modal.test.tsx`.
- Test `webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_gate.test.tsx`.
- Modify `webapp/channels/src/components/logged_in/logged_in.tsx` — mount the gate.

**Webapp admin console:**
- Modify `webapp/channels/src/components/admin_console/admin_definition.tsx` — add `type: 'bool'` entry under Site Configuration → Customization.

**i18n:**
- Modify `webapp/channels/src/i18n/en.json` — overlay strings + admin label/help strings.

---

## Task 1: Server config field + default

**Files:**
- Modify: `server/public/model/config.go` (struct near line 418; SetDefaults near line 874)
- Test: `server/public/model/config_test.go`

- [ ] **Step 1: Write the failing test**

Add to `server/public/model/config_test.go`:

```go
func TestServiceSettingsForceMobileAppDefault(t *testing.T) {
	s := ServiceSettings{}
	s.SetDefaults(false)
	require.NotNil(t, s.ForceMobileApp)
	require.False(t, *s.ForceMobileApp)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd server && go test ./public/model/ -run TestServiceSettingsForceMobileAppDefault -v`
Expected: FAIL — `s.ForceMobileApp undefined`.

- [ ] **Step 3: Add the struct field**

In `server/public/model/config.go`, in `type ServiceSettings struct`, immediately after the `EnableDesktopLandingPage` field (line ~418):

```go
	EnableDesktopLandingPage                          *bool
	ForceMobileApp                                    *bool   `access:"site_customization"`
```

- [ ] **Step 4: Add the default**

In `func (s *ServiceSettings) SetDefaults(isUpdate bool)`, immediately after the `EnableDesktopLandingPage` default block (line ~874-876):

```go
	if s.EnableDesktopLandingPage == nil {
		s.EnableDesktopLandingPage = NewPointer(true)
	}

	if s.ForceMobileApp == nil {
		s.ForceMobileApp = NewPointer(false)
	}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd server && go test ./public/model/ -run TestServiceSettingsForceMobileAppDefault -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add server/public/model/config.go server/public/model/config_test.go
git commit -m "feat(server): add ServiceSettings.ForceMobileApp config field"
```

---

## Task 2: Serialize ForceMobileApp into client config

**Files:**
- Modify: `server/config/client.go` (after line ~29, `EnableDesktopLandingPage`)
- Test: `server/config/client_test.go`

- [ ] **Step 1: Write the failing test**

Add to `server/config/client_test.go`:

```go
func TestGenerateClientConfigForceMobileApp(t *testing.T) {
	c := &model.Config{}
	c.SetDefaults()
	c.ServiceSettings.ForceMobileApp = model.NewPointer(true)

	props := GenerateClientConfig(c, "", nil)
	require.Equal(t, "true", props["ForceMobileApp"])
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd server && go test ./config/ -run TestGenerateClientConfigForceMobileApp -v`
Expected: FAIL — `props["ForceMobileApp"]` is empty, not `"true"`.

- [ ] **Step 3: Add the serialization line**

In `server/config/client.go`, inside `GenerateClientConfig`, immediately after the `EnableDesktopLandingPage` line (line ~29):

```go
	props["EnableDesktopLandingPage"] = strconv.FormatBool(*c.ServiceSettings.EnableDesktopLandingPage)
	props["ForceMobileApp"] = strconv.FormatBool(*c.ServiceSettings.ForceMobileApp)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd server && go test ./config/ -run TestGenerateClientConfigForceMobileApp -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add server/config/client.go server/config/client_test.go
git commit -m "feat(server): expose ForceMobileApp in client config"
```

---

## Task 3: Add webapp config types

**Files:**
- Modify: `webapp/platform/types/src/config.ts` (ClientConfig near line 72; AdminConfig ServiceSettings near line 430)

No standalone test (type-only change; verified by `tsc` in later tasks).

- [ ] **Step 1: Add to ClientConfig**

In `webapp/platform/types/src/config.ts`, immediately after the `EnableDesktopLandingPage: 'true' | 'false';` line (~72):

```typescript
    EnableDesktopLandingPage: 'true' | 'false';
    ForceMobileApp: 'true' | 'false';
```

- [ ] **Step 2: Add to AdminConfig ServiceSettings**

In the same file, immediately after the `EnableDesktopLandingPage: boolean;` line (~430):

```typescript
    EnableDesktopLandingPage: boolean;
    ForceMobileApp: boolean;
```

- [ ] **Step 3: Verify types compile**

Run: `cd webapp/platform/types && npx tsc --noEmit`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add webapp/platform/types/src/config.ts
git commit -m "feat(types): add ForceMobileApp to config types"
```

---

## Task 4: Presentational overlay component

**Files:**
- Create: `webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_modal.tsx`
- Create: `webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_modal.scss`
- Test: `webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_modal.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_modal.test.tsx`:

```typescript
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd webapp/channels && npm test -- force_mobile_app_modal.test`
Expected: FAIL — cannot resolve `./force_mobile_app_modal`.

- [ ] **Step 3: Create the component**

Create `webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_modal.tsx`:

```typescript
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
```

- [ ] **Step 4: Create the styles**

Create `webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_modal.scss`:

```scss
.ForceMobileAppModal {
    position: fixed;
    z-index: 10000;
    display: flex;
    width: 100vw;
    height: 100vh;
    align-items: center;
    justify-content: center;
    padding: 24px;
    background: rgba(0, 0, 0, 0.85);
    inset: 0;

    &__card {
        max-width: 420px;
        padding: 32px;
        border-radius: 12px;
        background: var(--center-channel-bg, #fff);
        color: var(--center-channel-color, #3d3c40);
        text-align: center;
    }

    &__title {
        margin-bottom: 16px;
        font-size: 22px;
        font-weight: 600;
    }

    &__text {
        margin-bottom: 8px;
    }

    &__server {
        margin-bottom: 24px;
        font-family: monospace;
        font-weight: 600;
        word-break: break-all;
    }

    &__links {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    &__store {
        display: inline-block;
        padding: 12px 16px;
        border-radius: 8px;
        background: var(--button-bg, #1c58d9);
        color: var(--button-color, #fff);
        font-weight: 600;
        text-decoration: none;

        &:hover {
            text-decoration: none;
            opacity: 0.9;
        }
    }
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd webapp/channels && npm test -- force_mobile_app_modal.test`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_modal.tsx webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_modal.scss webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_modal.test.tsx
git commit -m "feat(webapp): add ForceMobileAppModal overlay component"
```

---

## Task 5: Gate container + index

**Files:**
- Create: `webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_gate.tsx`
- Create: `webapp/channels/src/components/force_mobile_app_modal/index.ts`
- Test: `webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_gate.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_gate.test.tsx`:

```typescript
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd webapp/channels && npm test -- force_mobile_app_gate.test`
Expected: FAIL — cannot resolve `./force_mobile_app_gate`.

- [ ] **Step 3: Create the gate**

Create `webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_gate.tsx`:

```typescript
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
```

- [ ] **Step 4: Create the index re-export**

Create `webapp/channels/src/components/force_mobile_app_modal/index.ts`:

```typescript
export {default} from './force_mobile_app_gate';
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd webapp/channels && npm test -- force_mobile_app_gate.test`
Expected: PASS (all three cases).

- [ ] **Step 6: Commit**

```bash
git add webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_gate.tsx webapp/channels/src/components/force_mobile_app_modal/index.ts webapp/channels/src/components/force_mobile_app_modal/force_mobile_app_gate.test.tsx
git commit -m "feat(webapp): add ForceMobileAppGate container"
```

---

## Task 6: Mount the gate in LoggedIn layout

**Files:**
- Modify: `webapp/channels/src/components/logged_in/logged_in.tsx` (imports top; render near line 147)

- [ ] **Step 1: Add the import**

In `webapp/channels/src/components/logged_in/logged_in.tsx`, with the other component imports near the top (alongside the existing `LoadingScreen` import):

```typescript
import ForceMobileAppGate from 'components/force_mobile_app_modal';
```

- [ ] **Step 2: Render the gate alongside children**

In the `render()` method, replace the final return (line ~147):

```typescript
        return this.props.children;
```

with:

```typescript
        return (
            <>
                {this.props.children}
                <ForceMobileAppGate/>
            </>
        );
```

- [ ] **Step 3: Verify it compiles and existing tests pass**

Run: `cd webapp/channels && npx tsc --noEmit --project tsconfig.json 2>&1 | head -20`
Expected: no new errors referencing `logged_in` or `force_mobile_app_modal`.

Run: `cd webapp/channels && npm test -- logged_in`
Expected: existing LoggedIn tests still PASS.

- [ ] **Step 4: Commit**

```bash
git add webapp/channels/src/components/logged_in/logged_in.tsx
git commit -m "feat(webapp): mount ForceMobileAppGate in LoggedIn layout"
```

---

## Task 7: Admin console toggle

**Files:**
- Modify: `webapp/channels/src/components/admin_console/admin_definition.tsx` (Customization section, after `TeamSettings.EnableCustomBrand` entry near line 2331)

- [ ] **Step 1: Add the bool setting entry**

In `webapp/channels/src/components/admin_console/admin_definition.tsx`, immediately after the closing `},` of the `TeamSettings.EnableCustomBrand` entry (line ~2331), insert:

```typescript
                        {
                            type: 'bool',
                            key: 'ServiceSettings.ForceMobileApp',
                            label: defineMessage({id: 'admin.customization.forceMobileAppTitle', defaultMessage: 'Force Mobile App (block web for non-admins): '}),
                            help_text: defineMessage({id: 'admin.customization.forceMobileAppDesc', defaultMessage: 'When true, non-admin users opening the web client (browser or desktop app) are shown a full-screen prompt to download and use the mobile app, and cannot access chat from the web. System admins are unaffected.'}),
                            isDisabled: it.not(it.userHasWritePermissionOnResource(RESOURCE_KEYS.SITE.CUSTOMIZATION)),
                        },
```

- [ ] **Step 2: Verify it compiles**

Run: `cd webapp/channels && npx tsc --noEmit --project tsconfig.json 2>&1 | grep -i admin_definition | head`
Expected: no errors referencing admin_definition.

- [ ] **Step 3: Commit**

```bash
git add webapp/channels/src/components/admin_console/admin_definition.tsx
git commit -m "feat(webapp): add ForceMobileApp toggle to System Console"
```

---

## Task 8: i18n strings

**Files:**
- Modify: `webapp/channels/src/i18n/en.json`

- [ ] **Step 1: Add the strings**

In `webapp/channels/src/i18n/en.json`, add the following keys (keep the file's existing alphabetical ordering — place each key in its correct alphabetical position; the values are):

```json
    "admin.customization.forceMobileAppTitle": "Force Mobile App (block web for non-admins): ",
    "admin.customization.forceMobileAppDesc": "When true, non-admin users opening the web client (browser or desktop app) are shown a full-screen prompt to download and use the mobile app, and cannot access chat from the web. System admins are unaffected.",
    "force_mobile_app.title": "Access from a browser is restricted",
    "force_mobile_app.instructions": "To continue, please download the Mattermost mobile app and sign in using this server address:",
    "force_mobile_app.appStore": "Download on the App Store",
    "force_mobile_app.googlePlay": "Get it on Google Play",
```

- [ ] **Step 2: Verify i18n ordering/validity**

Run: `cd webapp/channels && npm run check-types 2>/dev/null; node -e "JSON.parse(require('fs').readFileSync('src/i18n/en.json','utf8')); console.log('en.json valid')"`
Expected: `en.json valid`.

If the repo has an i18n sort check (`make i18n-check` or `npm run i18n-extract`), run it and fix ordering if it complains.

- [ ] **Step 3: Commit**

```bash
git add webapp/channels/src/i18n/en.json
git commit -m "feat(i18n): add ForceMobileApp strings"
```

---

## Task 9: Full verification

- [ ] **Step 1: Run all new/affected tests**

Run:
```bash
cd server && go test ./public/model/ -run ForceMobileApp -v && go test ./config/ -run ForceMobileApp -v
cd ../webapp/channels && npm test -- force_mobile_app
```
Expected: all PASS.

- [ ] **Step 2: Lint the new webapp files**

Run: `cd webapp/channels && npx eslint src/components/force_mobile_app_modal/`
Expected: no errors.

- [ ] **Step 3: Manual smoke (optional, documented)**

1. Build/run server + webapp.
2. In System Console → Site Configuration → Customization, toggle **Force Mobile App** on.
3. As a non-admin user in a browser: full-screen overlay appears with both store links and the server URL; chat is not reachable.
4. As a system admin: no overlay.
5. Toggle off: overlay disappears on reload.

---

## Self-Review Notes

- **Spec coverage:** config field (T1) + default (T1) + client serialization (T2) + types (T3) + overlay (T4) + gate logic w/ admin exemption (T5) + mount in logged-in area (T6) + console toggle (T7) + i18n (T8). All spec sections mapped.
- **Type consistency:** config key `ForceMobileApp` used identically in Go (`ServiceSettings.ForceMobileApp`), client.go props (`"ForceMobileApp"`), types (`ClientConfig.ForceMobileApp`), gate selector (`config?.ForceMobileApp === 'true'`), and admin key (`ServiceSettings.ForceMobileApp`). Store-link constants `IOS_APP_STORE_LINK`/`ANDROID_PLAY_STORE_LINK` exported from the modal and reused in its test.
- **No platform detection:** intentional — mobile native app does not run this code (see spec "Ключевое наблюдение").
