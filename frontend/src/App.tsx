import { Button } from "@/components/ui/button";

function App() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center space-y-6">
        <h1 className="text-4xl font-bold tracking-tight">ChatSphere</h1>
        <p className="text-muted-foreground">
          Real-time anonymous chat rooms
        </p>
        <Button>Get Started</Button>
      </div>
    </div>
  );
}

export default App;
