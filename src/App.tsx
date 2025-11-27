import { Route, Switch } from "wouter";
import Layout from "./components/Layout";
import Insight from "./pages/Insight";
import Discover from "./pages/Discover";
import AgentPage from "./pages/AgentPage";
import { RefreshProvider } from "./contexts/RefreshContext";

const AppContent = () => {
  return (
    <Switch>
      <Route path="/" component={Insight} />
      <Route path="/p/discover" component={Discover} />
      <Route path="/agent" component={AgentPage} />
    </Switch>
  );
};

function App() {
  return (
    <RefreshProvider>
      <Layout>
        <AppContent />
      </Layout>
    </RefreshProvider>
  );
}

export default App;
