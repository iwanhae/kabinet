import React from "react";
import { Link, useLocation } from "wouter";
import { Moon, Sun } from "lucide-react";
import { useTheme } from "../contexts/ThemeContext";
import { useNavigation } from "../hooks/useNavigation";
import { TimeRangePicker } from "./TimeRangePicker";
import ScanCostBar from "./ScanCostBar";
import { IconButton, cx } from "../ui";
import styles from "./Layout.module.css";

interface LayoutProps {
  children: React.ReactNode;
}

const Layout: React.FC<LayoutProps> = ({ children }) => {
  const [location] = useLocation();
  const navigate = useNavigation();
  const { isDarkMode, toggleTheme } = useTheme();

  const menuItems = [
    { text: "Overview", path: "/", href: navigate({ page: "insight" }) },
    {
      text: "Namespaces",
      path: "/p/namespaces",
      href: navigate({ page: "namespaces" }),
    },
    {
      text: "Explore",
      path: "/p/discover",
      href: navigate({ page: "discover" }),
    },
    { text: "Agent", path: "/agent", href: navigate({ page: "agent" }) },
  ];

  return (
    <>
      <header className={styles.topbar}>
        <Link href={menuItems[0].href} className={styles.wordmark}>
          Kabinet
        </Link>

        <nav className={styles.nav} aria-label="Primary">
          {menuItems.map((item) => (
            <Link
              key={item.path}
              href={item.href}
              className={cx(
                styles.navLink,
                location === item.path && styles.navLinkActive,
              )}
            >
              {item.text}
            </Link>
          ))}
        </nav>

        <div className={styles.spacer} />

        <div className={styles.controls}>
          <TimeRangePicker />
          <IconButton
            label={isDarkMode ? "Switch to light mode" : "Switch to dark mode"}
            onClick={toggleTheme}
          >
            {isDarkMode ? <Sun size={16} /> : <Moon size={16} />}
          </IconButton>
        </div>
      </header>

      <main className={styles.main}>{children}</main>
      <ScanCostBar />
    </>
  );
};

export default Layout;
