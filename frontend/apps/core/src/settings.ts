import { setSettings } from '@fantastic-admin/settings'

export default setSettings({
  app: {
    routeMode: 'html5',
    routeBaseOn: 'frontend',
    dynamicTitle: true,
    mobile: true,
    account: {
      auth: false,
    },
    home: {
      enable: true,
      title: '仪表盘',
      fullPath: '/',
    },
    copyright: {
      enable: false,
    },
  },
  menu: {
    mode: 'single',
    mainMenuClickMode: 'jump',
    subMenuUniqueExpand: true,
    subMenuCollapse: false,
    subMenuCollapseButton: true,
  },
  topbar: {
    mode: 'static',
    tabbar: true,
    toolbar: true,
  },
  toolbar: {
    breadcrumb: true,
    menuSearch: {
      enable: true,
      hotkeys: true,
    },
    fullscreen: true,
    pageReload: true,
    colorScheme: true,
  },
  tabbar: {
    icon: true,
    hotkeys: true,
  },
  page: {
    progress: true,
  },
  theme: {
    colorScheme: 'light',
    radius: 0.5,
  },
})
