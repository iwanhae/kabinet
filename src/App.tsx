import { Route, Switch } from "wouter";
import Layout from "./components/Layout";
import Insight from "./pages/Insight";
import Discover from "./pages/Discover";

function App() {
  return (
    <Layout>
      <Switch>
        <Route path="/" component={Insight} />
        <Route path="/discover" component={Discover} />
      </Switch>
    </Layout>
  );
}

export default App;
