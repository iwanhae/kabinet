# Frontend Development Guide

Welcome to the Kube Event Analyzer frontend! This guide provides the necessary information to get you started with development.

## Tech Stack

- **Framework**: [React](https://reactjs.org/) with [Vite](https://vitejs.dev/)
- **Language**: [TypeScript](https://www.typescriptlang.org/)
- **UI Library**: [Material-UI (MUI)](https://mui.com/)
- **State Management**: [Zustand](https://github.com/pmndrs/zustand)
- **Date & Time**: [Day.js](https://day.js.org/) with `@mui/x-date-pickers`
- **Routing**: [Wouter](https://github.com/molefrog/wouter)
- **Styling**: [MUI's `styled` API](https://mui.com/system/styled/) & [Emotion](https://emotion.sh/)

## Getting Started

1.  **Install dependencies**:
    ```bash
    npm install
    ```
2.  **Run the development server**

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
├── components/     # Reusable React components (e.g., Layout, MetricCard, TimeRangePicker)
├── contexts/       # React contexts for global state (e.g., ThemeContext)
├── pages/          # Page components corresponding to routes (e.g., Insight, Discover)
├── stores/         # Zustand stores for global state management (e.g., timeRangeStore)
├── App.tsx         # Main application component, handles routing and global setup
├── main.tsx        # Application entry point
├── index.css       # Global styles
└── theme.ts        # MUI theme configuration (colors, typography, etc.)
```

---

## Core Concepts & Conventions

### 1. Global State: Time Range Management

A critical piece of global state is the **time range**, which dictates the query window for fetching and displaying event data. This state is managed by Zustand and is designed to be accessible from any component.

**Location**: `src/stores/timeRangeStore.ts`

**Core Idea**: The time range (`from`, `to`) is stored in a Zustand store and synchronized with the URL's query parameters. This ensures that the selected time range persists across page navigations and browser refreshes.

**How to Use**:

**A. Accessing the Current Time Range**

To get the current time range in any component, use the `useTimeRangeStore` hook. This provides direct access to the `from` and `to` values.

**Example (`src/pages/Discover.tsx`)**:

```tsx
import { useTimeRangeStore } from "../stores/timeRangeStore";

const Discover = () => {
  const { from, to } = useTimeRangeStore();

  useEffect(() => {
    // Fetch data whenever the time range changes
    console.log(`Querying data from ${from} to ${to}`);
    // fetchData(from, to);
  }, [from, to]);

  return (
    <div>
      Data for {from} - {to}
    </div>
  );
};
```

**B. Updating the Time Range**

You should **not** update the time range directly. Instead, use the `TimeRangePicker` component located in the main layout. It provides a user-friendly interface for selecting quick ranges (e.g., "Last 15 minutes") or absolute time frames.

The `TimeRangePicker` uses a custom hook, `useTimeRangeFromUrl`, which handles two key tasks:

1.  Updating the Zustand store.
2.  Updating the URL query parameters (`?from=...&to=...`).

This synchronization is crucial. The application's entry point (`App.tsx`) reads these URL parameters on initial load to set the store's state, ensuring persistence.

### 2. Styling

This project uses **Material-UI (MUI)** as its primary component library. For styling, we follow a specific convention to maintain consistency and readability.

**Primary Method: `styled` API**

For components with complex or reusable styles, always prefer using the `styled` utility from `@mui/material/styles`.

- **Why?** It keeps JSX clean, separates styling concerns, and makes styles reusable.
- **Where?** Define styled components at the top of the file they are used in.

**Secondary Method: `sx` Prop**

For simple, one-off styles that are not reused, it's acceptable to use the `sx` prop.

- **Why?** It's convenient for minor tweaks without the overhead of creating a new styled component.

**Theme & Theming**

- All theme-related values (colors, fonts, border-radius, etc.) are defined in `src/theme.ts`. We have separate configurations for `lightTheme` and `darkTheme`.
- The theme is provided to the entire application via `ThemeProvider` in `src/contexts/ThemeContext.tsx`.
- When creating styled components, always use theme tokens (e.g., `theme.palette.primary.main`) instead of hard-coded values.
- The dark/light mode toggle logic is managed within `ThemeContext`.

### 3. Routing

We use **Wouter** for client-side routing due to its minimal footprint and hook-based API.

- **Route Definitions**: All routes are defined in `src/App.tsx` using the `<Switch>` and `<Route>` components.
- **Navigation**:
  - To create navigation links, use the `<Link>` component from `wouter`.
  - For programmatic navigation, use the `useLocation` hook: `const [, setLocation] = useLocation(); setLocation('/new-path');`.

### 4. Creating New Components & Pages

- **Reusable Components**: If a component is used in multiple places (e.g., buttons, cards, inputs), create it inside `src/components/`.
- **Page Components**: A page is a component that maps directly to a route. Create new pages inside `src/pages/` and add the corresponding route to `App.tsx`.
- **Component Logic**: Keep components focused on their primary purpose. Separate complex logic into custom hooks if necessary.

---

By following these guidelines, we can ensure the frontend codebase remains clean, consistent, and easy to maintain. If you have any questions, please refer to the existing components as a reference.
