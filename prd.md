# ChatSphere — Product Requirements Document

## 1. Product Overview

**Product Name:** ChatSphere

**Description:** A real-time anonymous chat web application featuring public and private rooms. Users can create rooms, join conversations, and chat without creating an account. Designed to handle thousands of concurrent users over WebSocket connections.

**Scale Target:** Thousands of concurrent WebSocket connections.

---

## 2. Tech Stack

| Layer | Technology |
|---|---|
| **Backend** | Go — native WebSocket handling via `gorilla/websocket` or `nhooyr.io/websocket` |
| **Frontend** | React with shadcn/ui (Tailwind CSS-based component library) |
| **Storage** | Redis — message history, room state, user sessions |
| **Deployment** | Docker Compose (local development), Kubernetes (production) |

---

## 3. User Flows

### 3.1 Landing Page

- A single scrollable feed of public rooms.
- An "Enter Code" bar at the top of the page for joining private rooms by code.

### 3.2 Public Room List

- Sorted by **active user count** (most active rooms first).
- Each room card displays:
  - Room name
  - Active user count
  - Last message preview
  - Creator name

### 3.3 Room Creation

- Any user can create a room (public or private).
- **Rate limit:** Max 3 rooms per hour per IP address.
- Creation form fields:
  - Room name (required)
  - Description (optional)
  - Max capacity (configurable)
  - Public / Private toggle
- Private rooms receive an **auto-generated 6-character alphanumeric code** for sharing.

### 3.4 Joining a Room

- On entering a room, the user sees an **inline username input bar** above the chat area.
- Messages are visible immediately (read-only) before setting a username.
- The user **cannot send messages** until a username is set.
- Usernames are **per-room** — they are not persisted across rooms or sessions.

### 3.5 Chat Experience

- **Layout:** Slack-like — messages grouped by user, sidebar with online members, input bar at bottom.
- **Theme:** Dark theme only.
- **Responsive:** Fully responsive and mobile-friendly.

---

## 4. Message System

### 4.1 Message Types

- Rich text with **Markdown formatting** support.
- Clickable links.
- **Emoji picker** for inline emoji insertion.

### 4.2 Message History

- Full message persistence in Redis.
- On room join, load the **last 50 messages**.
- Scroll up to **lazy-load older messages** in batches.

### 4.3 System Messages

System messages are displayed inline in the chat for the following events:

- User joined
- User left
- User kicked
- User banned
- User muted
- Room expiration warnings

### 4.4 Typing Indicators

- Show `"[User] is typing..."` when a single user is typing.
- If **3 or more** users are typing, show `"Several people are typing..."`.

### 4.5 Message Rate Limiting

- Max **10 messages per 10 seconds** per user.

---

## 5. Room Lifecycle

### 5.1 Expiration Rules

A room expires and is cleaned up when either condition is met:

- **2 hours** since the last message in the room, OR
- **15 minutes** with no users connected.

### 5.2 Expiration Warnings

- System messages warn room participants before expiration occurs.

---

## 6. Moderation

The **room creator** acts as the room admin and has access to moderation controls.

### 6.1 Moderation Controls

- Accessible via **right-click or click on a username** (context menu).
- Available actions: Kick, Mute, Ban.

### 6.2 Kick

- User is removed from the room.
- Blocked from rejoining for **15 minutes**.

### 6.3 Ban

- User is blocked by **IP and session** for the lifetime of the room.

### 6.4 Mute

- Timed mute — the muted user sees `"You have been muted for X minutes"`.
- The chat input is **disabled** for the duration of the mute.

---

## 7. Connectivity

### 7.1 Reconnection

- **Automatic reconnect** on connection loss.
- Rejoin the room with the **same username**.
- **Backfill missed messages** that were sent during the disconnection.

### 7.2 Notifications

- **Visual only** — unread message count displayed in the **browser tab title** when the tab is inactive.

---

## 8. Non-Functional Requirements

- Handle **thousands of concurrent WebSocket connections**.
- **Responsive design** — mobile-first, built with shadcn/ui.
- **Dark theme only** — no light mode toggle.
