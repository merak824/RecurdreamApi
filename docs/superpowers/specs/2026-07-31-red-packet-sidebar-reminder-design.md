# Red Packet Sidebar Reminder Design

## Goal

Show a small unread indicator beside the user-facing red packet navigation item when a newly published active activity is available. Opening the red packet page marks that activity as seen in the current browser, so the indicator remains hidden until a later activity is published.

## Scope

- The reminder applies to the `/red-packets` navigation item for regular users and for the personal-account section shown to administrators.
- The `/admin/red-packets` management item does not receive this reminder.
- Seen state is local to the current browser and is not written to the backend.
- Existing red packet participation, eligibility, drawing, and balance behavior is unchanged.

## User Experience

- An unread activity is represented by an 8px red circular indicator aligned to the right side of the navigation item.
- The indicator has no continuous animation and does not change the navigation item's dimensions.
- It remains visible when the sidebar is collapsed.
- Screen readers receive an additional localized "new activity" label while the indicator is present.
- Opening `/red-packets` marks the current active activity as seen before the reminder is rendered, avoiding an unread flash on the destination page.

## State And Storage

- The current activity is identified by its stable numeric activity ID returned by `GET /red-packets/current`.
- Seen state is stored under a localStorage key scoped by the authenticated user ID, so accounts sharing one browser do not affect each other.
- The stored value is the last seen active activity ID.
- An activity is unread when a current active activity exists and its ID differs from the stored ID.
- No current activity, a failed request, or an unauthenticated user produces no indicator.
- A browser `storage` event updates other open tabs after one tab marks the activity as seen.

## Refresh Lifecycle

- Check the current activity when the reminder state starts.
- Recheck every 60 seconds while the application is mounted.
- Recheck when the browser window regains focus.
- Stop the interval and remove event listeners when the owning component is unmounted.
- Ignore request failures and preserve normal navigation behavior.

## Component Design

- Add a focused composable that owns fetching, localStorage comparison, cross-tab synchronization, and cleanup.
- `AppSidebar.vue` consumes the composable and exposes the unread state only on the `/red-packets` nav item.
- Navigation item rendering remains generic by adding an optional unread flag to the existing navigation item model.
- The red indicator uses existing Tailwind color tokens and dark-mode-compatible placement without adding a new icon dependency.

## Data Flow

1. The sidebar starts the reminder composable for the authenticated user.
2. The composable requests the current red packet activity.
3. If the current route is `/red-packets`, it stores the activity ID as seen and returns a read state.
4. Otherwise it compares the current activity ID with the user-scoped stored ID.
5. The sidebar renders the indicator when the comparison reports unread.
6. A later activity receives a new ID, causing the next refresh to show the indicator again.

## Testing

- Unit-test the composable with controlled activity responses, route changes, localStorage, focus, storage events, and timer cleanup.
- Verify the first active activity is unread when no seen value exists.
- Verify visiting `/red-packets` stores the current activity ID and hides the indicator.
- Verify a later activity ID becomes unread again.
- Verify seen values are isolated by authenticated user ID.
- Verify API failure produces no unread state and does not throw into the sidebar.
- Verify `AppSidebar.vue` renders the indicator only on the user-facing red packet item and includes accessible text.

## Non-Goals

- Cross-device synchronization.
- Push notifications, WebSocket delivery, email, or toast notifications.
- A numeric unread count.
- Reminder state for drafts, canceled activities, or the admin management menu.
