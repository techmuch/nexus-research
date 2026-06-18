# Getting Started with NEXUS Research Station

Welcome to the NEXUS Research Station! This guide will walk you through launching the system and making your first API calls.

## 🚀 Launching the Station

1. **Build the Application**:
   Ensure you have Go and Node.js installed, then compile both components from the root directory:
   ```bash
   make build
   ```

2. **Serve locally**:
   Start the compiled Go binary:
   ```bash
   ./bin/nexus-research serve --port 8080
   ```

3. **Open the browser**:
   Navigate to [http://localhost:8080](http://localhost:8080). You should see the workbench UI initialized with the system status panel.

---

## 🎨 Using the UI

The workspace is powered by `flexlayout-react` and is fully dynamic:
- **Drag and Drop**: Grab any tab header and drag it to split, stack, or reposition the viewports.
- **Theming**: Select theme variables from the configuration settings. Supporting Light, Dark, Retro WOPR, and Georgia Tech colors.
- **Command Menu**: Click the main menus to invoke registered commands such as **Ping API Status**.
