import React from "react";
import {
  Box,
  CssBaseline,
  AppBar,
  Toolbar,
  Typography,
  Drawer,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  IconButton,
  Chip,
} from "@mui/material";
import { styled, type Theme } from "@mui/material/styles";
import { Link, useLocation } from "wouter";
import InsightsIcon from "@mui/icons-material/Insights";
import TravelExploreIcon from "@mui/icons-material/TravelExplore";
import LightModeIcon from "@mui/icons-material/LightMode";
import DarkModeIcon from "@mui/icons-material/DarkMode";
import { useTheme } from "../contexts/ThemeContext";

const drawerWidth = 260;

const StyledAppBar = styled(AppBar)(({ theme }: { theme: Theme }) => ({
  zIndex: theme.zIndex.drawer + 1,
  boxShadow: "none",
}));

const StyledDrawer = styled(Drawer)({
  width: drawerWidth,
  flexShrink: 0,
  [`& .MuiDrawer-paper`]: {
    width: drawerWidth,
    boxSizing: "border-box",
    borderRight: "none",
  },
});

const MainContent = styled("main")(({ theme }: { theme: Theme }) => ({
  flexGrow: 1,
  padding: theme.spacing(3),
  backgroundColor: theme.palette.background.default,
  minHeight: "100vh",
}));

const StyledListItemButton = styled(ListItemButton)(({ theme }) => ({
  paddingTop: theme.spacing(1.5),
  paddingBottom: theme.spacing(1.5),
  paddingLeft: theme.spacing(3),
  paddingRight: theme.spacing(3),
  borderRadius: 3,
  "&.Mui-selected": {
    backgroundColor: theme.palette.primary.main,
    color: "white",
    "&:hover": {
      backgroundColor: theme.palette.primary.dark,
    },
    "& .MuiListItemIcon-root": {
      color: "white",
    },
  },
}));

interface LayoutProps {
  children: React.ReactNode;
}

const Layout: React.FC<LayoutProps> = ({ children }) => {
  const [location] = useLocation();
  const { isDarkMode, toggleTheme } = useTheme();

  const menuItems = [
    { text: "Insight", href: "/", icon: <InsightsIcon /> },
    { text: "Discover", href: "/discover", icon: <TravelExploreIcon /> },
  ];

  return (
    <Box sx={{ display: "flex" }}>
      <CssBaseline />
      <StyledAppBar position="fixed">
        <Toolbar sx={{ justifyContent: "space-between" }}>
          <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
            <Typography
              variant="h6"
              noWrap
              component="div"
              sx={{ fontWeight: 700 }}
            >
              Kube Event Analyzer
            </Typography>
            <Chip
              label="Analytics"
              size="small"
              sx={{
                backgroundColor: "primary.main",
                color: "white",
                fontWeight: 600,
                fontSize: "0.75rem",
              }}
            />
          </Box>
          <IconButton
            color="inherit"
            onClick={toggleTheme}
            sx={{
              borderRadius: 2,
              "&:hover": {
                backgroundColor: "rgba(29, 155, 240, 0.1)",
              },
            }}
          >
            {isDarkMode ? <LightModeIcon /> : <DarkModeIcon />}
          </IconButton>
        </Toolbar>
      </StyledAppBar>
      <StyledDrawer variant="permanent">
        <Toolbar />
        <Box sx={{ overflow: "auto", p: 1 }}>
          <List sx={{ pt: 2 }}>
            {menuItems.map((item) => (
              <ListItem key={item.text} disablePadding sx={{ mb: 1 }}>
                <Link
                  href={item.href}
                  style={{ textDecoration: "none", width: "100%" }}
                >
                  <StyledListItemButton selected={location === item.href}>
                    <ListItemIcon sx={{ minWidth: 40 }}>
                      {item.icon}
                    </ListItemIcon>
                    <ListItemText
                      primary={item.text}
                      slotProps={{
                        primary: {
                          fontWeight: location === item.href ? 700 : 500,
                          fontSize: "1rem",
                        },
                      }}
                    />
                  </StyledListItemButton>
                </Link>
              </ListItem>
            ))}
          </List>
        </Box>
      </StyledDrawer>
      <MainContent>
        <Toolbar />
        {children}
      </MainContent>
    </Box>
  );
};

export default Layout;
