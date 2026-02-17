import { NavLink } from 'react-router-dom'

interface SidebarProps {
  projectKey: string
}

const navItems = [
  { to: '', label: 'Overview', end: true },
  { to: 'items', label: 'Items', end: false },
  { to: 'queues', label: 'Queues', end: false },
  { to: 'milestones', label: 'Milestones', end: false },
]

export function Sidebar({ projectKey }: SidebarProps) {
  const base = `/projects/${projectKey}`

  return (
    <nav className="w-48 shrink-0">
      <ul className="space-y-1">
        {navItems.map((item) => (
          <li key={item.to}>
            <NavLink
              to={`${base}/${item.to}`}
              end={item.end}
              className={({ isActive }) =>
                `block px-3 py-2 rounded-md text-sm font-medium ${
                  isActive
                    ? 'bg-indigo-50 text-indigo-700'
                    : 'text-gray-700 hover:bg-gray-100'
                }`
              }
            >
              {item.label}
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  )
}
