import { Route, Switch } from "wouter";
import Layout from "./components/Layout";
import Insight from "./pages/Insight";
import Discover from "./pages/Discover";

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
    <Layout>
      <AppContent />
    </Layout>
  );
}

export default App;
