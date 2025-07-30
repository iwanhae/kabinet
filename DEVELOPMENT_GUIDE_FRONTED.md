# Frontend Development Guide

Welcome to the Kube Event Analyzer frontend! This guide provides the necessary information to get you started with development.

## Tech Stack

- **Framework**: [React](https://reactjs.org/) with [Vite](https://vitejs.dev/)
- **Language**: [TypeScript](https://www.typescriptlang.org/)
- **UI Library**: [Material-UI (MUI)](https://mui.com/)
- **Routing**: [Wouter](https://github.com/molefrog/wouter)
- **Styling**: [MUI's `styled` API](https://mui.com/system/styled/) & [Emotion](https://emotion.sh/)

## Getting Started

1.  **Install dependencies**:
    ```bash
    npm install
    ```
2.  **Run the development server**:

    ```bash
    npm run dev
    ```

    The application will be available at `http://localhost:5173`.

3.  **Linting**:
    ```bash
    npm run lint
    ```

## Project Structure

The `src` directory is organized as follows:

```
src/
├── assets/         # Static assets like images and SVGs
├── components/     # Reusable React components (e.g., Layout, MetricCard)
├── contexts/       # React contexts for global state (e.g., ThemeContext)
├── pages/          # Page components corresponding to routes (e.g., Insight, Discover)
├── App.tsx         # Main application component, handles routing
├── main.tsx        # Application entry point
├── index.css       # Global styles
└── theme.ts        # MUI theme configuration (colors, typography, etc.)
```

---

## Core Concepts & Conventions

### 1. Styling

This project uses **Material-UI (MUI)** as its primary component library. For styling, we follow a specific convention to maintain consistency and readability.

**Primary Method: `styled` API**

For components with complex or reusable styles, always prefer using the `styled` utility from `@mui/material/styles`.

- **Why?** It keeps JSX clean, separates styling concerns, and makes styles reusable.
- **Where?** Define styled components at the top of the file they are used in.

**Example (`src/components/MetricCard.tsx`)**:

```tsx
import { Card, styled } from "@mui/material";

const StyledCard = styled(Card)({
  height: "100%",
  // ... more styles
});

const MetricCard = () => {
  return <StyledCard>{/* ... */}</StyledCard>;
};
```

**Secondary Method: `sx` Prop**

For simple, one-off styles that are not reused, it's acceptable to use the `sx` prop.

- **Why?** It's convenient for minor tweaks without the overhead of creating a new styled component.

**Example**:

```tsx
<Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>{/* ... */}</Box>
```

**Theme & Theming**

- All theme-related values (colors, fonts, border-radius, etc.) are defined in `src/theme.ts`. We have separate configurations for `lightTheme` and `darkTheme`.
- The theme is provided to the entire application via `ThemeProvider` in `src/contexts/ThemeContext.tsx`.
- When creating styled components, always use theme tokens (e.g., `theme.palette.primary.main`) instead of hard-coded values.
- The dark/light mode toggle logic is managed within `ThemeContext`.

### 2. Routing

We use **Wouter** for client-side routing due to its minimal footprint and hook-based API.

- **Route Definitions**: All routes are defined in `src/App.tsx` using the `<Switch>` and `<Route>` components.
- **Navigation**:
  - To create navigation links, use the `<Link>` component from `wouter`.
  - For programmatic navigation, use the `useLocation` hook: `const [, setLocation] = useLocation(); setLocation('/new-path');`.

**Example (`src/App.tsx`)**:

```tsx
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
```

### 3. Creating New Components & Pages

- **Reusable Components**: If a component is used in multiple places (e.g., buttons, cards, inputs), create it inside `src/components/`.
- **Page Components**: A page is a component that maps directly to a route. Create new pages inside `src/pages/` and add the corresponding route to `App.tsx`.
- **Component Logic**: Keep components focused on their primary purpose. Separate complex logic into custom hooks if necessary.

---

By following these guidelines, we can ensure the frontend codebase remains clean, consistent, and easy to maintain. If you have any questions, please refer to the existing components as a reference.
