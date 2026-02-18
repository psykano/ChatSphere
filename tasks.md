## Tasks

### Project Setup
- [x] Initialize Go backend module and project structure
- [x] Initialize React frontend with Vite and shadcn/ui
- [x] Set up Docker Compose with Go, React, and Redis services
- [x] Configure Tailwind CSS with dark theme only

### Backend — WebSocket Infrastructure
- [x] Set up WebSocket server using gorilla/websocket or nhooyr.io/websocket
- [x] Implement connection manager for concurrent WebSocket connections
- [x] Implement automatic reconnection and session resumption
- [x] Implement missed message backfill on reconnect

### Backend — Room System
- [x] Create room creation endpoint (name, description, capacity, public/private)
- [x] Generate 6-character alphanumeric codes for private rooms
- [x] Create room listing endpoint sorted by active user count
- [x] Implement room join and leave logic
- [x] Implement room expiration (2hr no messages or 15min no users)
- [x] Send system message warnings before room expiration
- [x] Rate limit room creation to 3 per hour per IP

### Backend — Message System
- [x] Implement message sending and broadcasting over WebSocket
- [x] Persist messages in Redis
- [x] Load last 50 messages on room join
- [x] Implement lazy-loading older messages in batches
- [x] Rate limit messages to 10 per 10 seconds per user
- [x] Implement system messages (join, leave, kick, ban, mute, expiration)
- [x] Implement typing indicator broadcasting

### Backend — User System
- [x] Implement anonymous user sessions
- [x] Implement per-room username setting
- [x] Track online users per room

### Backend — Moderation
- [x] Implement kick (remove user, block rejoin for 15 minutes)
- [x] Implement ban (block by IP and session for room lifetime)
- [x] Implement timed mute with duration

### Frontend — Landing Page
- [x] Build scrollable public room list feed
- [x] Build room card component (name, active users, last message, creator)
- [x] Build "Enter Code" bar for joining private rooms

### Frontend — Room Creation
- [x] Build room creation form (name, description, capacity, public/private toggle)
- [x] Display auto-generated private room code after creation

### Frontend — Chat UI
- [x] Build Slack-like chat layout (messages area, sidebar, input bar)
- [x] Implement dark theme styling
- [x] Make layout fully responsive and mobile-friendly
- [x] Build inline username input bar above chat area
- [x] Implement read-only mode before username is set
- [x] Render Markdown formatting in messages
- [x] Make links clickable in messages
- [x] Add emoji picker for inline emoji insertion
- [x] Display typing indicators ("User is typing..." / "Several people are typing...")
- [x] Group messages by user

### Frontend — Connectivity
- [x] Establish and manage WebSocket connection
- [x] Implement automatic reconnection on connection loss
- [x] Backfill missed messages after reconnect
- [x] Show unread message count in browser tab title when inactive

### Frontend — Moderation UI
- [x] Build context menu on username click/right-click
- [x] Add kick, mute, and ban actions to context menu
- [x] Show "You have been muted for X minutes" message and disable input

### Deployment
- [x] Finalize Docker Compose configuration for local development
- [x] Create Kubernetes manifests for production deployment
