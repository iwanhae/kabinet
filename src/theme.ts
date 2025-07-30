import { createTheme, type ThemeOptions } from "@mui/material/styles";

declare module "@mui/material/styles" {
  interface Palette {
    twitter: {
      main: string;
      light: string;
      dark: string;
    };
  }

  interface PaletteOptions {
    twitter?: {
      main: string;
      light: string;
      dark: string;
    };
  }
}

const baseTheme: ThemeOptions = {
  typography: {
    fontFamily: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif',
    h4: {
      fontWeight: 700,
      fontSize: "1.75rem",
    },
    h5: {
      fontWeight: 600,
      fontSize: "1.5rem",
    },
    h6: {
      fontWeight: 600,
      fontSize: "1.25rem",
    },
    body1: {
      fontSize: "0.875rem",
    },
    body2: {
      fontSize: "0.75rem",
    },
  },
  shape: {
    borderRadius: 8,
  },
  components: {
    MuiCard: {
      styleOverrides: {
        root: {
          boxShadow: "none",
          border: "1px solid",
        },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: "none",
          fontWeight: 600,
          borderRadius: 8,
        },
      },
    },
  },
};

export const darkTheme = createTheme({
  ...baseTheme,
  palette: {
    mode: "dark",
    primary: {
      main: "#1d9bf0",
      light: "#4dabf5",
      dark: "#1a8cd8",
    },
    twitter: {
      main: "#1d9bf0",
      light: "#4dabf5",
      dark: "#1a8cd8",
    },
    background: {
      default: "#000000",
      paper: "#16181c",
    },
    text: {
      primary: "#e7e9ea",
      secondary: "#71767b",
    },
    divider: "#2f3336",
    success: {
      main: "#00ba7c",
    },
    error: {
      main: "#f4212e",
    },
    warning: {
      main: "#ffad1f",
    },
  },
  components: {
    ...baseTheme.components,
    MuiCard: {
      styleOverrides: {
        root: {
          backgroundColor: "#16181c",
          borderColor: "#2f3336",
          boxShadow: "none",
          border: "1px solid #2f3336",
        },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        root: {
          backgroundColor: "rgba(0, 0, 0, 0.8)",
          backdropFilter: "blur(12px)",
          borderBottom: "1px solid #2f3336",
        },
      },
    },
    MuiDrawer: {
      styleOverrides: {
        paper: {
          backgroundColor: "#000000",
          borderRight: "1px solid #2f3336",
        },
      },
    },
    MuiListItemButton: {
      styleOverrides: {
        root: {
          borderRadius: 8,
          margin: "2px 8px",
          "&:hover": {
            backgroundColor: "rgba(29, 155, 240, 0.1)",
          },
          "&.Mui-selected": {
            backgroundColor: "rgba(29, 155, 240, 0.1)",
            "&:hover": {
              backgroundColor: "rgba(29, 155, 240, 0.15)",
            },
          },
        },
      },
    },
  },
});

export const lightTheme = createTheme({
  ...baseTheme,
  palette: {
    mode: "light",
    primary: {
      main: "#1d9bf0",
      light: "#4dabf5",
      dark: "#1a8cd8",
    },
    twitter: {
      main: "#1d9bf0",
      light: "#4dabf5",
      dark: "#1a8cd8",
    },
    background: {
      default: "#ffffff",
      paper: "#ffffff",
    },
    text: {
      primary: "#0f1419",
      secondary: "#536471",
    },
    divider: "#eff3f4",
    success: {
      main: "#00ba7c",
    },
    error: {
      main: "#f4212e",
    },
    warning: {
      main: "#ffad1f",
    },
  },
  components: {
    ...baseTheme.components,
    MuiCard: {
      styleOverrides: {
        root: {
          backgroundColor: "#ffffff",
          borderColor: "#eff3f4",
          boxShadow: "none",
          border: "1px solid #eff3f4",
        },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        root: {
          backgroundColor: "rgba(255, 255, 255, 0.8)",
          backdropFilter: "blur(12px)",
          borderBottom: "1px solid #eff3f4",
          color: "#0f1419",
        },
      },
    },
    MuiDrawer: {
      styleOverrides: {
        paper: {
          backgroundColor: "#ffffff",
          borderRight: "1px solid #eff3f4",
        },
      },
    },
    MuiListItemButton: {
      styleOverrides: {
        root: {
          borderRadius: 8,
          margin: "2px 8px",
          "&:hover": {
            backgroundColor: "rgba(29, 155, 240, 0.1)",
          },
          "&.Mui-selected": {
            backgroundColor: "rgba(29, 155, 240, 0.1)",
            "&:hover": {
              backgroundColor: "rgba(29, 155, 240, 0.15)",
            },
          },
        },
      },
    },
  },
});
