import Tippy from '@tippyjs/react'
import 'tippy.js/dist/tippy.css'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'

interface NodeActionButtonsProps {
  onEdit?: () => void
  onEmit?: () => void
}

export const NodeActionButtons = ({ onEdit, onEmit }: NodeActionButtonsProps) => {
  return (
    <div
      className="action-buttons absolute z-50 text-sm -top-10 left-1/2 transform -translate-x-1/2 flex bg-white dark:bg-zinc-800 shadow-lg rounded-lg border border-gray-200 dark:border-zinc-700"
      onClick={(e) => e.stopPropagation()}
    >
      <Tippy content="Manually emit an event" placement="top" theme="dark" arrow>
        <button
          onClick={(e) => {
            e.stopPropagation()
            onEmit?.()
          }}
          className="flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-l-md transition-colors"
        >
          <MaterialSymbol name="send" size="sm" />
        </button>
      </Tippy>

      <div className="w-px h-8 bg-gray-300 dark:bg-zinc-600" />

      <Tippy content="Edit" placement="top" theme="dark" arrow>
        <button
          onClick={(e) => {
            e.stopPropagation()
            onEdit?.()
          }}
          className="flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-r-md transition-colors"
        >
          <MaterialSymbol name="edit" size="sm" />
        </button>
      </Tippy>
    </div>
  )
}
