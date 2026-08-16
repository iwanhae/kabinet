import { Route, Switch } from "wouter";
import { SWRConfig } from "swr";
import Layout from "./components/Layout";
import Overview from "./pages/Overview";
import Namespaces from "./pages/Namespaces";
import Nodes from "./pages/Nodes";
import Components from "./pages/Components";
import Explore from "./pages/Explore";
import McpPage from "./pages/McpPage";
import { RefreshProvider } from "./contexts/RefreshContext";

const swrConfig = {
  revalidateOnFocus: false,
  keepPreviousData: true,
  dedupingInterval: 5000,
  errorRetryCount: 2,
};

const AppContent = () => {
  return (
    <Switch>
      <Route path="/" component={Overview} />
      <Route path="/p/namespaces" component={Namespaces} />
      <Route path="/p/nodes" component={Nodes} />
      <Route path="/p/components" component={Components} />
      <Route path="/p/discover" component={Explore} />
      <Route path="/p/mcp" component={McpPage} />
    </Switch>
  );
};

function App() {
  return (
    <SWRConfig value={swrConfig}>
      <RefreshProvider>
        <Layout>
          <AppContent />
        </Layout>
      </RefreshProvider>
    </SWRConfig>
  );
}

export default App;
