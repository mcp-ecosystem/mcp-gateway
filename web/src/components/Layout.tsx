import React from 'react';
import {
  Navbar,
  NavbarContent,
  NavbarItem,
  Button,
  Link as HeroLink,
  Avatar,
  Tooltip,
  Dropdown,
  DropdownTrigger,
  DropdownMenu,
  DropdownItem
} from "@heroui/react";
import { Icon } from '@iconify/react';
import { useLocation, Link, useNavigate } from 'react-router-dom';

interface LayoutProps {
  children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const [isCollapsed, setIsCollapsed] = React.useState(false);
  const [isDark, setIsDark] = React.useState(() => {
    const savedTheme = localStorage.getItem('theme');
    return savedTheme === 'dark';
  });

  // Initialize theme on mount
  React.useEffect(() => {
    if (isDark) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, []);

  const menuItems = [
    { to: "/", icon: "lucide:server", label: "Gateway Manager" },
    { to: "/chat", icon: "lucide:message-square", label: "MCP Chat" }
  ];

  const handleLogout = () => {
    window.localStorage.removeItem('token');
    navigate('/login');
  };

  const toggleTheme = () => {
    setIsDark(!isDark);
    document.documentElement.classList.toggle('dark');
    localStorage.setItem('theme', !isDark ? 'dark' : 'light');
  };

  return (
    <div className="min-h-screen bg-background text-foreground">
      {/* Top Navigation Bar */}
      <Navbar
        className="bg-card border-b border-border shadow-sm"
        maxWidth="full"
        height="4rem"
      >
        <NavbarContent justify="end" className="gap-4">
          <NavbarItem>
            <Tooltip content="Join Discord">
              <Button
                as={HeroLink}
                href="https://discord.gg/udf69cT9TY"
                target="_blank"
                variant="light"
                isIconOnly
              >
                <Icon icon="ic:baseline-discord" className="text-2xl" />
              </Button>
            </Tooltip>
          </NavbarItem>
          <NavbarItem>
            <Tooltip content="View on GitHub">
              <Button
                as={HeroLink}
                href="https://github.com/mcp-ecosystem/mcp-gateway"
                target="_blank"
                variant="light"
                isIconOnly
              >
                <Icon icon="mdi:github" className="text-2xl" />
              </Button>
            </Tooltip>
          </NavbarItem>
          <NavbarItem>
            <Tooltip content={`Switch to ${isDark ? "light" : "dark"} mode`}>
              <Button
                variant="light"
                isIconOnly
                onPress={toggleTheme}
              >
                <Icon
                  icon={isDark ? "lucide:sun" : "lucide:moon"}
                  className="text-2xl"
                />
              </Button>
            </Tooltip>
          </NavbarItem>
        </NavbarContent>
      </Navbar>

      <div className="flex h-[calc(100vh-4rem)]">
        {/* Sidebar */}
        <div
          className={`h-screen bg-card text-foreground flex flex-col fixed left-0 top-0 z-40 transition-all duration-300 border-r border-border shadow-lg ${
            isCollapsed ? "w-20" : "w-64"
          }`}
        >
          <div className="flex items-center justify-between p-4 border-b border-border h-16">
            {!isCollapsed && <span className="text-xl font-bold">MCP Admin</span>}
            <Button
              isIconOnly
              variant="light"
              onPress={() => setIsCollapsed(!isCollapsed)}
              aria-label="Toggle sidebar"
            >
              <Icon icon={isCollapsed ? "lucide:chevron-right" : "lucide:chevron-left"} />
            </Button>
          </div>

          <nav className="flex-1 overflow-y-auto p-2">
            {menuItems.map((item) => (
              isCollapsed ? (
                <Tooltip
                  key={item.to}
                  content={item.label}
                  placement="right"
                >
                  <Link
                    to={item.to}
                    className={`flex items-center w-full px-4 py-2 rounded-lg mb-1 ${
                      (item.to === "/"
                        ? location.pathname === "/"
                        : location.pathname.startsWith(item.to))
                        ? 'bg-primary/10 text-primary'
                        : 'hover:bg-accent text-foreground'
                    }`}
                  >
                    <Icon icon={item.icon} className="text-xl" />
                  </Link>
                </Tooltip>
              ) : (
                <Link
                  key={item.to}
                  to={item.to}
                  className={`flex items-center w-full px-4 py-2 rounded-lg mb-1 ${
                    (item.to === "/"
                      ? location.pathname === "/"
                      : location.pathname.startsWith(item.to))
                      ? 'bg-primary/10 text-primary'
                      : 'hover:bg-accent text-foreground'
                  }`}
                >
                  <Icon icon={item.icon} className="text-xl" />
                  <span className="ml-2">{item.label}</span>
                </Link>
              )
            ))}
          </nav>

          <div className="border-t border-border p-4">
            <Dropdown placement="top-end">
              <DropdownTrigger>
                <div className="flex items-center gap-3 cursor-pointer hover:bg-accent p-2 rounded-lg transition-colors">
                  <Avatar
                    icon={<Icon icon="lucide:user" className="text-xl" />}
                    name="Admin"
                    size="sm"
                    color="primary"
                  />
                  {!isCollapsed && (
                    <div className="flex-1">
                      <p className="text-sm font-semibold text-foreground">Admin</p>
                      <p className="text-xs text-muted-foreground">Administrator</p>
                    </div>
                  )}
                </div>
              </DropdownTrigger>
              <DropdownMenu aria-label="User Actions">
                <DropdownItem
                  key="profile"
                  startContent={<Icon icon="lucide:user" className="text-xl" />}
                >
                  个人信息
                </DropdownItem>
                <DropdownItem
                  key="settings"
                  startContent={<Icon icon="lucide:settings" className="text-xl" />}
                >
                  系统设置
                </DropdownItem>
                <DropdownItem
                  key="logout"
                  className="text-destructive"
                  color="danger"
                  startContent={<Icon icon="lucide:log-out" className="text-xl" />}
                  onPress={handleLogout}
                >
                  退出登录
                </DropdownItem>
              </DropdownMenu>
            </Dropdown>
          </div>
        </div>

        {/* Main Content */}
        <main className={`flex-1 overflow-auto transition-all duration-300 ${isCollapsed ? "ml-20" : "ml-64"} bg-background text-foreground p-6`}>
          {children}
        </main>
      </div>
    </div>
  );
} 