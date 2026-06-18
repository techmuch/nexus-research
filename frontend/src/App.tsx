import { useEffect, useState } from 'react';
import { 
  ShellLayout, 
  initializeShell, 
  commandRegistry, 
  menuRegistry, 
  componentRegistry,
  useLayoutStore 
} from 'nexus-shell';

interface StatusResponse {
  status: string;
  uptime: string;
  version: string;
  db_connected: boolean;
}

function DashboardTab() {
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchStatus = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/status');
      const data = await res.json();
      setStatus(data);
    } catch (err) {
      console.error('Failed to fetch API status:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStatus();
  }, []);

  return (
    <div className="p-6 bg-card text-card-foreground h-full overflow-y-auto font-sans flex flex-col gap-6">
      <div className="border-b border-border pb-4 flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold tracking-tight text-primary">NEXUS Research Station</h2>
          <p className="text-sm text-muted-foreground mt-1">Autonomous Multi-Agent Orchestration & Analysis Workbench</p>
        </div>
        <button 
          onClick={fetchStatus}
          disabled={loading}
          className="px-4 py-2 bg-primary text-primary-foreground font-semibold rounded hover:bg-primary/95 transition-all text-sm shadow-sm"
        >
          {loading ? 'Refreshing...' : 'Refresh Status'}
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-muted/50 border border-border/80 rounded-lg p-5 flex flex-col justify-between">
          <span className="text-xs font-semibold tracking-wider text-muted-foreground uppercase">System State</span>
          <span className="text-2xl font-extrabold text-foreground mt-2">ACTIVE</span>
          <span className="text-xs text-muted-foreground mt-4">All core engines online</span>
        </div>

        <div className="bg-muted/50 border border-border/80 rounded-lg p-5 flex flex-col justify-between">
          <span className="text-xs font-semibold tracking-wider text-muted-foreground uppercase">Go API Connection</span>
          <span className="text-2xl font-extrabold text-green-500 mt-2">
            {status ? 'CONNECTED' : 'DISCONNECTED'}
          </span>
          <span className="text-xs text-muted-foreground mt-4">
            {status ? `Uptime: ${status.uptime}` : 'Checking server connectivity...'}
          </span>
        </div>

        <div className="bg-muted/50 border border-border/80 rounded-lg p-5 flex flex-col justify-between">
          <span className="text-xs font-semibold tracking-wider text-muted-foreground uppercase">Build Version</span>
          <span className="text-2xl font-extrabold text-foreground mt-2">{status?.version || '0.1.0'}</span>
          <span className="text-xs text-muted-foreground mt-4">NEXUS-shell Engine v0.1.19</span>
        </div>
      </div>

      <div className="bg-muted/30 border border-border/80 rounded-lg p-6 flex flex-col gap-4">
        <h3 className="text-lg font-semibold text-foreground border-b border-border/50 pb-2">Active Research Pipelines</h3>
        <ul className="divide-y divide-border/50">
          <li className="py-3 flex justify-between items-center">
            <div>
              <p className="font-semibold text-foreground text-sm">LLM Agent Evaluation Pipeline</p>
              <p className="text-xs text-muted-foreground">Contextual feedback loops, trajectory analysis</p>
            </div>
            <span className="px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-500/15 text-emerald-600 dark:text-emerald-400">Running</span>
          </li>
          <li className="py-3 flex justify-between items-center">
            <div>
              <p className="font-semibold text-foreground text-sm">Visual Argumentation Mapper</p>
              <p className="text-xs text-muted-foreground">Compendium-style dialogue map rendering</p>
            </div>
            <span className="px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-500/15 text-amber-600 dark:text-amber-400">Idle</span>
          </li>
        </ul>
      </div>
    </div>
  );
}

function App() {
  useEffect(() => {
    // 1. Initialize core registries
    initializeShell();

    // 2. Register Custom Tab Component
    componentRegistry.register('nexus-dashboard', DashboardTab);

    // 3. Register Custom commands
    commandRegistry.registerCommand({
      id: 'app.ping_api',
      label: 'Ping Backend API',
      execute: async () => {
        try {
          const res = await fetch('/api/status');
          const data = await res.json();
          alert(`API Response:\n\nStatus: ${data.status}\nUptime: ${data.uptime}\nVersion: ${data.version}\nDatabase connected: ${data.db_connected}`);
        } catch (err) {
          alert(`API Error: Failed to connect to server.`);
        }
      },
    });

    // 4. Add to the File menu
    menuRegistry.registerMenu('File', {
      id: 'file.ping_api',
      label: 'Ping API Status',
      commandId: 'app.ping_api',
    });

    // 5. Automatically open the dashboard tab on startup
    const layoutState = useLayoutStore.getState();
    layoutState.addTab('nexus-dashboard', 'Dashboard');

  }, []);

  return (
    <div style={{ width: '100vw', height: '100vh' }}>
      <ShellLayout title={<div className="font-bold text-lg text-primary">NEXUS RESEARCH STATION</div>} />
    </div>
  );
}

export default App;
