import { RoomCard } from "@/components/room-card";
import { useRooms } from "@/hooks/use-rooms";

function App() {
  const { rooms, loading, error } = useRooms();

  return (
    <div className="mx-auto flex min-h-screen max-w-2xl flex-col px-4 py-8">
      <header className="mb-8 text-center">
        <h1 className="text-4xl font-bold tracking-tight">ChatSphere</h1>
        <p className="mt-2 text-muted-foreground">
          Real-time anonymous chat rooms
        </p>
      </header>

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
              <RoomCard key={room.id} room={room} />
            ))}
          </div>
        )}
      </main>
    </div>
  );
}

export default App;
