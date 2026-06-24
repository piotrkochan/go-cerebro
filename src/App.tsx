import { RouterProvider } from '@tanstack/react-router';

import { router } from './app/router';

export default function App() {
  return <RouterProvider router={router} />;
}
