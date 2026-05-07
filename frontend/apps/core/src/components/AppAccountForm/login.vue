<script setup lang="ts">
import { toTypedSchema } from '@vee-validate/zod'
import { useForm } from 'vee-validate'
import * as z from 'zod'
import { FormControl, FormField, FormItem, FormMessage } from '@/ui/shadcn/ui/form'

defineOptions({
  name: 'LoginForm',
})

const props = defineProps<{
  account?: string
}>()

const emits = defineEmits<{
  onLogin: [account?: string]
}>()

const appAccountStore = useAppAccountStore()

const title = import.meta.env.VITE_APP_TITLE
const loading = ref(false)


const form = useForm({
  validationSchema: toTypedSchema(z.object({
    account: z.string().min(1, '请输入用户名'),
    password: z.string().min(1, '请输入密码'),
    remember: z.boolean(),
  })),
  initialValues: {
    account: props.account ?? localStorage.getItem('login_account') ?? '',
    password: '',
    remember: localStorage.getItem('login_account') !== null,
  },
})
const onSubmit = form.handleSubmit((values) => {
  loading.value = true
  appAccountStore.login(values).then(() => {
    if (values.remember) {
      localStorage.setItem('login_account', values.account)
    }
    else {
      localStorage.removeItem('login_account')
    }
    emits('onLogin', values.account)
  }).finally(() => {
    loading.value = false
  })
})

</script>

<template>
  <div class="p-12 flex-col-stretch-center min-h-500px w-full">
    <div class="mb-6 space-y-2">
      <h3 class="text-4xl font-bold">
        欢迎使用 👋🏻
      </h3>
      <p class="text-sm text-muted-foreground lg:text-base">
        {{ title }}
      </p>
    </div>
    <div class="mb-4">
      <div class="inline-flex rounded-lg border bg-muted/40 px-3 py-2 text-sm text-muted-foreground">
        账号密码登录
      </div>
    </div>
    <div>
      <form @submit="onSubmit">
        <FormField v-slot="{ componentField, errors }" name="account">
          <FormItem class="pb-6 relative space-y-0">
            <FormControl>
              <FaInput type="text" placeholder="用户名" class="w-full" :class="{ 'border-destructive': errors.length }" v-bind="componentField">
                <template #start>
                  <FaIcon name="i-lucide:user" />
                </template>
              </FaInput>
            </FormControl>
            <Transition enter-active-class="transition-opacity" enter-from-class="opacity-0" leave-active-class="transition-opacity" leave-to-class="opacity-0">
              <FormMessage class="text-xs bottom-1 absolute" />
            </Transition>
          </FormItem>
        </FormField>
        <FormField v-slot="{ componentField, errors }" name="password">
          <FormItem class="pb-6 relative space-y-0">
            <FormControl>
              <FaInput type="password" placeholder="密码" class="w-full" :class="{ 'border-destructive': errors.length }" v-bind="componentField">
                <template #start>
                  <FaIcon name="i-lucide:lock" />
                </template>
              </FaInput>
            </FormControl>
            <Transition enter-active-class="transition-opacity" enter-from-class="opacity-0" leave-active-class="transition-opacity" leave-to-class="opacity-0">
              <FormMessage class="text-xs bottom-1 absolute" />
            </Transition>
          </FormItem>
        </FormField>
        <div class="mb-4 flex-center-between">
          <div class="flex-center-start">
            <FormField v-slot="{ componentField }" type="checkbox" name="remember">
              <FormItem>
                <FormControl>
                  <FaCheckbox v-bind="componentField">
                    记住我
                  </FaCheckbox>
                </FormControl>
              </FormItem>
            </FormField>
          </div>
          <span class="text-sm text-secondary-foreground op-50">忘记密码请联系管理员</span>
        </div>
        <FaButton :loading="loading" size="lg" class="w-full" type="submit">
          登录
        </FaButton>
        <div class="text-sm mt-4 text-center text-secondary-foreground op-60">
          使用 OpsManage 管理员账号登录
        </div>
      </form>
    </div>

  </div>
</template>
