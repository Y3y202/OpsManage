import api from '../index'

export default {
  // 后端获取路由数据：OpsManage 使用前端静态路由
  routeList: () => Promise.resolve({ data: [] }),

  // 登录
  async login(data: {
    account: string
    password: string
  }) {
    const res = await api.post('/api/auth/login', {
      username: data.account,
      password: data.password,
    })
    return {
      data: {
        account: res.data.user?.username || data.account,
        token: res.data.token,
        avatar: '',
      },
    }
  },

  // 获取权限
  permission: () => Promise.resolve({ data: { permissions: ['*'] } }),

  // 修改密码
  passwordEdit: (data: {
    password: string
    newPassword: string
  }) => api.post('/api/profile/password', {
    old_password: data.password,
    new_password: data.newPassword,
  }),
}
