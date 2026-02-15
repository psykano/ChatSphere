## Tasks

### Project Setup
- [ ] Initialize Go backend module and project structure
- [ ] Initialize React frontend with Vite and shadcn/ui
- [ ] Set up Docker Compose with Go, React, and Redis services
- [ ] Configure Tailwind CSS with dark theme only

### Backend — WebSocket Infrastructure
- [ ] Set up WebSocket server using gorilla/websocket or nhooyr.io/websocket
- [ ] Implement connection manager for concurrent WebSocket connections
- [ ] Implement automatic reconnection and session resumption
- [ ] Implement missed message backfill on reconnect

### Backend — Room System
- [ ] Create room creation endpoint (name, description, capacity, public/private)
- [ ] Generate 6-character alphanumeric codes for private rooms
- [ ] Create room listing endpoint sorted by active user count
- [ ] Implement room join and leave logic
- [ ] Implement room expiration (2hr no messages or 15min no users)
- [ ] Send system message warnings before room expiration
- [ ] Rate limit room creation to 3 per hour per IP

### Backend — Message System
- [ ] Implement message sending and broadcasting over WebSocket
- [ ] Persist messages in Redis
- [ ] Load last 50 messages on room join
- [ ] Implement lazy-loading older messages in batches
- [ ] Rate limit messages to 10 per 10 seconds per user
- [ ] Implement system messages (join, leave, kick, ban, mute, expiration)
- [ ] Implement typing indicator broadcasting

### Backend — User System
- [ ] Implement anonymous user sessions
- [ ] Implement per-room username setting
- [ ] Track online users per room

### Backend — Moderation
- [ ] Implement kick (remove user, block rejoin for 15 minutes)
- [ ] Implement ban (block by IP and session for room lifetime)
- [ ] Implement timed mute with duration

### Frontend — Landing Page
- [ ] Build scrollable public room list feed
- [ ] Build room card component (name, active users, last message, creator)
- [ ] Build "Enter Code" bar for joining private rooms

### Frontend — Room Creation
- [ ] Build room creation form (name, description, capacity, public/private toggle)
- [ ] Display auto-generated private room code after creation

### Frontend — Chat UI
- [ ] Build Slack-like chat layout (messages area, sidebar, input bar)
- [ ] Implement dark theme styling
- [ ] Make layout fully responsive and mobile-friendly
- [ ] Build inline username input bar above chat area
- [ ] Implement read-only mode before username is set
- [ ] Render Markdown formatting in messages
- [ ] Make links clickable in messages
- [ ] Add emoji picker for inline emoji insertion
- [ ] Display typing indicators ("User is typing..." / "Several people are typing...")
- [ ] Group messages by user

### Frontend — Connectivity
- [ ] Establish and manage WebSocket connection
- [ ] Implement automatic reconnection on connection loss
- [ ] Backfill missed messages after reconnect
- [ ] Show unread message count in browser tab title when inactive

### Frontend — Moderation UI
- [ ] Build context menu on username click/right-click
- [ ] Add kick, mute, and ban actions to context menu
- [ ] Show "You have been muted for X minutes" message and disable input

### Deployment
- [ ] Finalize Docker Compose configuration for local development
- [ ] Create Kubernetes manifests for production deployment
