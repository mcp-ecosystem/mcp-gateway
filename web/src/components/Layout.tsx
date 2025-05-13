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
import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation, Link, useNavigate } from 'react-router-dom';

import { getCurrentUser } from '../api/auth';
import { toast } from '../utils/toast';


import { ChangePasswordDialog } from './ChangePasswordDialog';
import { LanguageSwitcher } from './LanguageSwitcher';
import { WechatQRCode } from './WechatQRCode';

interface LayoutProps {
  children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [isCollapsed, setIsCollapsed] = React.useState(false);
  const [isDark, setIsDark] = React.useState(() => {
    const savedTheme = window.localStorage.getItem('theme');
    return savedTheme === 'dark';
  });
  const [isChangePasswordOpen, setIsChangePasswordOpen] = React.useState(false);
  const [isWechatQRCodeOpen, setIsWechatQRCodeOpen] = React.useState(false);
  const [userInfo, setUserInfo] = useState<{ username: string; role: string } | null>(null);

  // Initialize theme on mount
  React.useEffect(() => {
    if (isDark) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, [isDark]);

  useEffect(() => {
    const fetchUserInfo = async () => {
      try {
        const response = await getCurrentUser();
        setUserInfo(response.data);
      } catch (error) {
        toast.error(t('errors.fetch_user_info', { error: (error as Error).message }), {
          duration: 3000,
        });
      }
    };

    fetchUserInfo();
  }, [t]);

  const menuItems = [
    {
      key: 'chat',
      label: t('nav.chat'),
      icon: 'lucide:message-square',
      path: '/chat',
    },
    {
      key: 'gateway',
      label: t('nav.gateway'),
      icon: 'lucide:server',
      path: '/gateway',
    },
    ...(userInfo?.role === 'admin' ? [
      {
        key: 'users',
        label: t('nav.users'),
        icon: 'lucide:users',
        path: '/users',
      },
      {
        key: 'tenants',
        label: t('nav.tenants'),
        icon: 'lucide:building',
        path: '/tenants',
      }
    ] : []),
  ];

  const handleLogout = () => {
    window.localStorage.removeItem('token');
    navigate('/login');
  };

  const toggleTheme = () => {
    setIsDark(!isDark);
    document.documentElement.classList.toggle('dark');
    window.localStorage.setItem('theme', !isDark ? 'dark' : 'light');
  };

  return (
    <div className="min-h-screen bg-background text-foreground">
      {/* Top Navigation Bar */}
      <Navbar
        className="bg-card border-b border-border shadow-sm"
        maxWidth="full"
        height="4rem"
      >
        <NavbarContent className={`transition-all duration-300 ${isCollapsed ? 'ml-20' : 'ml-64'}`}>
          <NavbarItem>
            <Button
              isIconOnly
              variant="light"
              onPress={() => setIsCollapsed(!isCollapsed)}
              aria-label={t('common.toggle_sidebar')}
            >
              <Icon icon={isCollapsed ? "lucide:chevron-right" : "lucide:chevron-left"} />
            </Button>
          </NavbarItem>
        </NavbarContent>
        <NavbarContent justify="end" className="gap-4">
          <NavbarItem>
            <LanguageSwitcher />
          </NavbarItem>
          <NavbarItem>
            <Tooltip content={t('common.join_wechat')}>
              <Button
                variant="light"
                isIconOnly
                onPress={() => setIsWechatQRCodeOpen(true)}
              >
                <Icon icon="mdi:wechat" className="text-2xl" />
              </Button>
            </Tooltip>
          </NavbarItem>
          <NavbarItem>
            <Tooltip content={t('common.join_discord')}>
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
            <Tooltip content={t('common.view_github')}>
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
            <Tooltip content={t('common.view_docs')}>
              <Button
                as={HeroLink}
                href="https://mcp.ifuryst.com/"
                target="_blank"
                variant="light"
                isIconOnly
              >
                <Icon icon="mdi:book-open-page-variant" className="text-2xl" />
              </Button>
            </Tooltip>
          </NavbarItem>
          <NavbarItem>
            <Tooltip content={t('common.switch_theme', { theme: isDark ? t('common.light') : t('common.dark') })}>
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
          <div className="flex items-center justify-center p-4 border-b border-border h-16">
            {isCollapsed ? (
              <img src="/logo.png" alt="MCP Logo" className="w-8 h-8" />
            ) : (
              <div className="flex items-center gap-2">
                <img src="/logo.png" alt="MCP Logo" className="w-6 h-6" />
                <span className="text-xl font-bold">MCP Admin</span>
              </div>
            )}
          </div>

          <nav className="flex-1 overflow-y-auto p-2">
            {menuItems.map((item) => (
              isCollapsed ? (
                <Tooltip
                  key={item.path}
                  content={item.label}
                  placement="right"
                >
                  <Link
                    to={item.path}
                    className={`flex items-center w-full px-4 py-2 rounded-lg mb-1 ${
                      (item.path === "/"
                        ? location.pathname === "/"
                        : location.pathname.startsWith(item.path))
                        ? 'bg-primary/10 text-primary'
                        : 'hover:bg-accent text-foreground'
                    }`}
                  >
                    <Icon icon={item.icon} className="text-xl" />
                  </Link>
                </Tooltip>
              ) : (
                <Link
                  key={item.path}
                  to={item.path}
                  className={`flex items-center w-full px-4 py-2 rounded-lg mb-1 ${
                    (item.path === "/"
                      ? location.pathname === "/"
                      : location.pathname.startsWith(item.path))
                      ? 'bg-primary/10 text-primary'
                      : 'hover:bg-accent text-foreground'
                  }`}
                >
                  <Icon icon={item.icon} className="text-xl mr-3" />
                  <span>{item.label}</span>
                </Link>
              )
            ))}
          </nav>

          {/* User Profile Section */}
          <div className="p-4 border-t border-border">
            <Dropdown placement="top-end">
              <DropdownTrigger>
                {isCollapsed ? (
                  <Button
                    isIconOnly
                    variant="light"
                    className="w-full"
                  >
                    <Avatar
                      size="sm"
                      name={userInfo?.username || 'User'}
                      className="bg-primary/10"
                    />
                  </Button>
                ) : (
                  <Button
                    variant="light"
                    className="w-full flex items-center justify-start gap-2"
                  >
                    <Avatar
                      size="sm"
                      name={userInfo?.username || 'User'}
                      className="bg-primary/10"
                    />
                    <div className="flex flex-col items-start">
                      <span className="text-sm font-medium">{userInfo?.username || 'User'}</span>
                      <span className="text-xs text-muted-foreground">{userInfo?.role || 'User'}</span>
                    </div>
                  </Button>
                )}
              </DropdownTrigger>
              <DropdownMenu aria-label="User menu">
                <DropdownItem
                  key="change-password"
                  startContent={<Icon icon="lucide:key" />}
                  onPress={() => setIsChangePasswordOpen(true)}
                >
                  {t('auth.change_password')}
                </DropdownItem>
                <DropdownItem
                  key="logout"
                  startContent={<Icon icon="lucide:log-out" />}
                  onPress={handleLogout}
                  className="text-danger"
                >
                  {t('auth.logout')}
                </DropdownItem>
              </DropdownMenu>
            </Dropdown>
          </div>
        </div>

        {/* Main Content */}
        <div className={`flex-1 transition-all duration-300 ${isCollapsed ? 'ml-20' : 'ml-64'}`}>
          <div className="p-6">
            {children}
          </div>
        </div>
      </div>

      <ChangePasswordDialog
        isOpen={isChangePasswordOpen}
        onOpenChange={() => setIsChangePasswordOpen(false)}
      />

      <WechatQRCode
        isOpen={isWechatQRCodeOpen}
        onOpenChange={() => setIsWechatQRCodeOpen(false)}
      />
    </div>
  );
}
