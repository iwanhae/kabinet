# Frontend Development Guide

Welcome to the Kube Event Analyzer frontend! This guide provides the necessary information to get you started with development.

## Tech Stack

- **Framework**: [React](https://reactjs.org/) with [Vite](https://vitejs.dev/)
- **Language**: [TypeScript](https://www.typescriptlang.org/)
- **UI Library**: [Material-UI (MUI)](https://mui.com/)
- **Data Fetching**: [SWR](https://swr.vercel.app/)
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
├── components/     # Reusable React components
├── contexts/       # React contexts for global state
├── hooks/          # Custom React hooks (e.g., for data fetching)
├── pages/          # Page components corresponding to routes
├── stores/         # Zustand stores for global state management
├── App.tsx         # Main application component, handles routing and global setup
├── main.tsx        # Application entry point
├── index.css       # Global styles
└── theme.ts        # MUI theme configuration
```

---

## Core Concepts & Conventions

### 1. Data Fetching with `useEventsQuery`

All data fetching from the backend API should be handled by the `useEventsQuery` custom hook. This hook abstracts away the complexities of data fetching, caching, and state management, allowing components to focus solely on displaying the data.

**Location**: `src/hooks/useEventsQuery.ts`

**Core Idea**: The hook takes a SQL query string as an argument and automatically combines it with the current global time range from the Zustand store. It uses **SWR** to handle caching, revalidation, and managing loading/error states.

**How to Use**:

To fetch data, import the hook and pass your SQL query. The hook is **generic**, meaning you must provide a type or interface that describes the shape of the expected result objects. This provides full type safety for your data.

**Example (`src/pages/Insight.tsx`)**:

```tsx
import React from "react";
import { useEventsQuery } from "../hooks/useEventsQuery";
import { CircularProgress, Alert, Paper, Typography } from "@mui/material";

// 1. Define the type for the expected data objects
interface ReasonCount {
  reason: string;
  count: number;
}

const TopReasonsChart = () => {
  // 2. Define the SQL query
  const query =
    "SELECT reason, COUNT(*) as count FROM $events WHERE type = 'Warning' GROUP BY reason ORDER BY count DESC LIMIT 10";

  // 3. Call the hook with the type and query
  const { data, error, isLoading } = useEventsQuery<ReasonCount>(query);

  if (isLoading) return <CircularProgress />;
  if (error)
    return <Alert severity="error">Failed to load data: {error.message}</Alert>;

  return (
    <Paper>
      <Typography variant="h6">Top 10 Warning Reasons</Typography>
      <ul>
        {/* 4. 'data' is now fully typed as 'ReasonCount[] | undefined' */}
        {data?.map((item) => (
          <li key={item.reason}>
            {item.reason}: {item.count}
          </li>
        ))}
      </ul>
    </Paper>
  );
};
```

**Conditional Fetching**: If you pass `null` as the query, the hook will not trigger a request. This is useful when you need to wait for some condition to be met before fetching data.

### 2. Global State: Time Range Management

A critical piece of global state is the **time range**, which is automatically used by the `useEventsQuery` hook.

**Location**: `src/stores/timeRangeStore.ts`

**Core Idea**: The time range (`from`, `to`) is stored in a Zustand store and synchronized with the URL's query parameters. This ensures that the selected time range persists across page navigations and browser refreshes. Any component using `useEventsQuery` will automatically re-fetch data when the time range changes.

**How to Update**: You should **not** update the time range directly. Instead, use the `TimeRangePicker` component located in the main layout. It provides a user-friendly interface for selecting quick ranges or absolute time frames and handles updating both the Zustand store and the URL parameters.

### 3. Styling

This project uses **Material-UI (MUI)** as its primary component library. For styling, we follow a specific convention to maintain consistency and readability.

- **Primary Method: `styled` API**: For components with complex or reusable styles, always prefer using the `styled` utility from `@mui/material/styles`.
- **Secondary Method: `sx` Prop**: For simple, one-off styles, it's acceptable to use the `sx` prop.
- **Theme**: All theme-related values (colors, fonts, etc.) are defined in `src/theme.ts`. Always use theme tokens (e.g., `theme.palette.primary.main`) in styled components.

### 4. Routing

We use **Wouter** for client-side routing due to its minimal footprint and hook-based API.

- **Route Definitions**: All routes are defined in `src/App.tsx`.
- **Navigation**: Use the `<Link>` component for links and the `useLocation` hook for programmatic navigation.

### 5. Creating New Components & Pages

- **Reusable Components**: Create inside `src/components/`.
- **Page Components**: Create inside `src/pages/` and add the corresponding route to `App.tsx`.
- **Component Logic**: Keep components focused. Separate complex logic into custom hooks (like `useEventsQuery`).

---

By following these guidelines, we can ensure the frontend codebase remains clean, consistent, and easy to maintain.
