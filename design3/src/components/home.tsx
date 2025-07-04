import { Navbar } from './lib/Navbar/navbar'
import { Sidebar } from './lib/Sidebar/sidebar'
import { StackedLayout } from './lib/StackedLayout/stacked-layout'

function Example({ children }: { children: React.ReactNode }) {
  return (
    <StackedLayout
      navbar={<Navbar>{/* Your navbar content */}</Navbar>}
      sidebar={<Sidebar>{/* Your sidebar content */}</Sidebar>}
    >
      {/* Your page content */}
    </StackedLayout>
  )
}