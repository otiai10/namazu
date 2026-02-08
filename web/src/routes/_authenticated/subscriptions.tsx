import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/subscriptions')({
  beforeLoad: () => {
    throw redirect({ to: '/' })
  },
})
