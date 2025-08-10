import React from "react";
import {
  Box,
  CssBaseline,
  AppBar,
  Toolbar,
  Typography,
  IconButton,
  Tab,
  Tabs,
} from "@mui/material";
import { styled, type Theme } from "@mui/material/styles";
import { Link, useLocation } from "wouter";
import InsightsIcon from "@mui/icons-material/Insights";
import TravelExploreIcon from "@mui/icons-material/TravelExplore";
import LightModeIcon from "@mui/icons-material/LightMode";
import DarkModeIcon from "@mui/icons-material/DarkMode";
import { useTheme } from "../contexts/ThemeContext";
import { TimeRangePicker } from "./TimeRangePicker";
import { useNavigation } from "../hooks/useNavigation";

const StyledAppBar = styled(AppBar)(({ theme }: { theme: Theme }) => ({
  boxShadow: "none",
  borderBottom: `1px solid ${theme.palette.divider}`,
}));

const MainContent = styled("main")(({ theme }: { theme: Theme }) => ({
  padding: theme.spacing(3),
  backgroundColor: theme.palette.background.default,
  minHeight: "100vh",
  paddingTop: theme.spacing(12), // AppBar 높이만큼 여백 추가
}));

const StyledTabs = styled(Tabs)(({ theme }) => ({
  minHeight: 48,
  "& .MuiTab-root": {
    minHeight: 48,
    textTransform: "none",
    fontWeight: 600,
    fontSize: "1rem",
    "&.Mui-selected": {
      color: theme.palette.primary.main,
    },
  },
  "& .MuiTabs-indicator": {
    backgroundColor: theme.palette.primary.main,
    height: 3,
  },
}));

interface LayoutProps {
  children: React.ReactNode;
}

const Layout: React.FC<LayoutProps> = ({ children }) => {
  const [location] = useLocation();
  const navigate = useNavigation();
  const { isDarkMode, toggleTheme } = useTheme();

  const menuItems = [
    {
      text: "Insight",
      href: navigate({ page: "insight" }),
      icon: <InsightsIcon />,
    },
    {
      text: "Discover",
      href: navigate({
        page: "discover",
        params: {
          where: "1=1",
        },
      }),
      icon: <TravelExploreIcon />,
    },
  ];

  // 현재 경로에 맞는 탭 인덱스 계산
  const currentTabIndex = menuItems.findIndex(
    (item) => item.href.split("?")[0] === location,
  );

  return (
    <Box>
      <CssBaseline />
      <StyledAppBar position="fixed">
        <Toolbar sx={{ justifyContent: "space-between", minHeight: 64 }}>
          {/* 왼쪽: 브랜드와 네비게이션 */}
          <Box sx={{ display: "flex", alignItems: "center", gap: 3 }}>
            <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
              <Typography
                variant="h6"
                noWrap
                component="div"
                sx={{
                  fontWeight: 700,
                  color: "primary.main",
                }}
              >
                Kabinet
              </Typography>
            </Box>

            {/* 탭 네비게이션 */}
            <StyledTabs
              value={currentTabIndex >= 0 ? currentTabIndex : 0}
              sx={{ ml: 2 }}
            >
              {menuItems.map((item) => (
                <Tab
                  key={item.text}
                  label={item.text}
                  icon={item.icon}
                  iconPosition="start"
                  component={Link}
                  href={item.href}
                  sx={{
                    minWidth: 120,
                    gap: 1,
                  }}
                />
              ))}
            </StyledTabs>
          </Box>

          {/* 오른쪽: 시간 선택기와 테마 토글 */}
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <TimeRangePicker />
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
          </Box>
        </Toolbar>
      </StyledAppBar>

      {/* 메인 콘텐츠 - 이제 전체 너비 사용 */}
      <MainContent>{children}</MainContent>
    </Box>
  );
};

export default Layout;
