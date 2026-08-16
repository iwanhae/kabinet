import { lazy, Suspense } from "react";
import { Route, Switch } from "wouter";
import { SWRConfig } from "swr";
import Layout from "./components/Layout";
import Overview from "./pages/Overview";
import Namespaces from "./pages/Namespaces";
import Explore from "./pages/Explore";
import { RefreshProvider } from "./contexts/RefreshContext";
import { Spinner } from "./ui";

// The agent page pulls in the OpenAI SDK and markdown renderer — split it out.
const AgentPage = lazy(() => import("./pages/AgentPage"));

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
      <Route path="/p/discover" component={Explore} />
      <Route path="/agent" component={AgentPage} />
    </Switch>
  );
};

function App() {
  return (
    <SWRConfig value={swrConfig}>
      <RefreshProvider>
        <Layout>
          <Suspense
            fallback={
              <div
                style={{
                  display: "flex",
                  justifyContent: "center",
                  padding: 48,
                }}
              >
                <Spinner />
              </div>
            }
          >
            <AppContent />
          </Suspense>
        </Layout>
      </RefreshProvider>
    </SWRConfig>
  );
}

export default App;
