import { Route, Switch } from "wouter";
import Layout from "./components/Layout";
import Insight from "./pages/Insight";
import Discover from "./pages/Discover";
import { RefreshProvider } from "./contexts/RefreshContext";

const AppContent = () => {
  return (
    <Switch>
      <Route path="/" component={Insight} />
      <Route path="/discover" component={Discover} />
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
