import { useEffect } from "react";
import { Route, Switch, useLocation } from "wouter";
import Layout from "./components/Layout";
import Insight from "./pages/Insight";
import Discover from "./pages/Discover";
import { useTimeRangeStore } from "./stores/timeRangeStore";

const AppContent = () => {
  const { setTimeRange } = useTimeRangeStore();
  const [location] = useLocation();

  useEffect(() => {
    const searchParams = new URLSearchParams(window.location.search);
    const from = searchParams.get("from");
    const to = searchParams.get("to");

    if (from && to) {
      setTimeRange({ from, to });
    }
  }, [location, setTimeRange]);

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
