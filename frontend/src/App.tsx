import { useState } from "react";
import { ChatLayout } from "@/components/chat-layout";
import { CreateRoomForm } from "@/components/create-room-form";
import { EnterCodeBar } from "@/components/enter-code-bar";
import { RoomCard } from "@/components/room-card";
import type { Room } from "@/components/room-card";
import { RoomCodeDialog } from "@/components/room-code-dialog";
import { useCreateRoom } from "@/hooks/use-create-room";
import { useJoinByCode } from "@/hooks/use-join-by-code";
import { useRooms } from "@/hooks/use-rooms";

function App() {
  const { rooms, loading, error, refresh } = useRooms();
  const { joinByCode, loading: joining, error: joinError } = useJoinByCode();
  const { createRoom, loading: creating, error: createError } = useCreateRoom();
  const [createdRoom, setCreatedRoom] = useState<Room | null>(null);
  const [activeRoom, setActiveRoom] = useState<Room | null>(null);

  if (activeRoom) {
    return (
      <ChatLayout
        room={activeRoom}
        onLeave={() => {
          setActiveRoom(null);
          refresh();
        }}
      />
    );
  }

  return (
    <div className="mx-auto flex min-h-screen max-w-2xl flex-col px-4 py-8">
      <header className="mb-8 text-center">
        <h1 className="text-4xl font-bold tracking-tight">ChatSphere</h1>
        <p className="mt-2 text-muted-foreground">
          Real-time anonymous chat rooms
        </p>
      </header>

      <section className="mb-6" aria-label="Create a room">
        <CreateRoomForm
          onSubmit={async (input) => {
            const room = await createRoom(input);
            if (room) {
              refresh();
              if (!room.public && room.code) {
                setCreatedRoom(room);
              } else {
                setActiveRoom(room);
              }
            }
          }}
          loading={creating}
          error={createError}
        />
      </section>

      <section className="mb-6" aria-label="Join private room">
        <EnterCodeBar
          onJoin={async (code) => {
            const room = await joinByCode(code);
            if (room) {
              setActiveRoom(room);
            }
          }}
          loading={joining}
          error={joinError}
        />
      </section>

      <main className="flex-1">
        {loading && rooms.length === 0 && (
          <p className="text-center text-muted-foreground">Loading rooms...</p>
        )}

        {error && rooms.length === 0 && (
          <p className="text-center text-destructive-foreground">{error}</p>
        )}

        {!loading && !error && rooms.length === 0 && (
          <p className="text-center text-muted-foreground">
            No public rooms yet. Create one to get started!
          </p>
        )}

        {rooms.length > 0 && (
          <div className="space-y-3 overflow-y-auto">
            {rooms.map((room) => (
              <RoomCard
                key={room.id}
                room={room}
                onClick={setActiveRoom}
              />
            ))}
          </div>
        )}
      </main>

      {createdRoom?.code && (
        <RoomCodeDialog
          roomName={createdRoom.name}
          code={createdRoom.code}
          onClose={() => {
            const room = createdRoom;
            setCreatedRoom(null);
            setActiveRoom(room);
          }}
        />
      )}
    </div>
  );
}

export default App;
