import { useState } from 'react'
import { PanelLeftClose, Menu } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { ItemGroup, Item, ItemMedia, ItemContent, ItemTitle, ItemDescription } from '@/components/ui/item'
import { resolveIcon } from '@/lib/utils'
import { getColorClass } from '@/utils/colors'

export interface BuildingBlock {
  name: string
  label?: string
  description?: string
  type: 'trigger' | 'component' | 'blueprint'
  outputChannels?: { name: string }[]
  configuration?: any[]
  icon?: string
  color?: string
  id?: string // for blueprints
}

export interface BuildingBlocksSidebarProps {
  isOpen: boolean
  onToggle: (open: boolean) => void
  triggers: BuildingBlock[]
  components: BuildingBlock[]
  blueprints: BuildingBlock[]
  onBlockClick: (block: BuildingBlock) => void
}

export function BuildingBlocksSidebar({
  isOpen,
  onToggle,
  triggers,
  components,
  blueprints,
  onBlockClick,
}: BuildingBlocksSidebarProps) {
  const [activeTab, setActiveTab] = useState<'triggers' | 'components' | 'blueprints'>('triggers')

  if (!isOpen) {
    return (
      <Button
        variant="outline"
        size="icon"
        onClick={() => onToggle(true)}
        aria-label="Open sidebar"
        className="absolute top-4 left-4 z-10 shadow-md"
      >
        <Menu size={24} />
      </Button>
    )
  }

  return (
    <div className="w-96 h-full bg-white dark:bg-zinc-900 border-r border-zinc-200 dark:border-zinc-800 flex flex-col">
      {/* Sidebar Header with Tabs */}
      <div className="flex items-center gap-3 px-4 pt-4 pb-0">
        <Button
          variant="outline"
          size="icon"
          onClick={() => onToggle(false)}
          aria-label="Close sidebar"
        >
          <PanelLeftClose size={24} />
        </Button>
        <Tabs
          value={activeTab}
          onValueChange={(value) => setActiveTab(value as 'triggers' | 'components' | 'blueprints')}
          className="flex-1"
        >
          <TabsList className="w-full">
            <TabsTrigger value="triggers" className="flex-1">
              Triggers
            </TabsTrigger>
            <TabsTrigger value="components" className="flex-1">
              Components
            </TabsTrigger>
            <TabsTrigger value="blueprints" className="flex-1">
              Blueprints
            </TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      {/* Tab Content */}
      <div className="flex-1 overflow-hidden px-4">
        <Tabs
          value={activeTab}
          onValueChange={(value) => setActiveTab(value as 'triggers' | 'components' | 'blueprints')}
          className="flex-1 flex flex-col h-full"
        >
          <TabsContent
            value="triggers"
            className="flex-1 overflow-y-auto text-left mt-4 data-[state=inactive]:hidden"
          >
            <div className="!text-xs text-gray-500 dark:text-zinc-400 mb-3">
              Click on a trigger to add it to your workflow
            </div>
            <ItemGroup>
              {triggers.map((block) => {
                const IconComponent = resolveIcon(block.icon || 'zap')
                const colorClass = getColorClass(block.color)

                return (
                  <Item
                    key={`${block.type}-${block.name}`}
                    onClick={() => onBlockClick(block)}
                    className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                    size="sm"
                  >
                    <ItemMedia>
                      <IconComponent size={24} className={colorClass} />
                    </ItemMedia>
                    <ItemContent>
                      <ItemTitle>{block.label || block.name}</ItemTitle>
                      {block.description && (
                        <ItemDescription>{block.description}</ItemDescription>
                      )}
                    </ItemContent>
                  </Item>
                )
              })}
            </ItemGroup>
          </TabsContent>

          <TabsContent
            value="components"
            className="flex-1 overflow-y-auto text-left mt-4 data-[state=inactive]:hidden"
          >
            <div className="!text-xs text-gray-500 dark:text-zinc-400 mb-3">
              Click on a component to add it to your workflow
            </div>
            <ItemGroup>
              {components.map((block) => {
                const IconComponent = resolveIcon(block.icon || 'boxes')
                const colorClass = getColorClass(block.color)

                return (
                  <Item
                    key={`${block.type}-${block.name}`}
                    onClick={() => onBlockClick(block)}
                    className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                    size="sm"
                  >
                    <ItemMedia>
                      <IconComponent size={24} className={colorClass} />
                    </ItemMedia>
                    <ItemContent>
                      <ItemTitle>{block.label || block.name}</ItemTitle>
                      {block.description && (
                        <ItemDescription>{block.description}</ItemDescription>
                      )}
                    </ItemContent>
                  </Item>
                )
              })}
            </ItemGroup>
          </TabsContent>

          <TabsContent
            value="blueprints"
            className="flex-1 overflow-y-auto text-left mt-4 data-[state=inactive]:hidden"
          >
            <div className="!text-xs text-gray-500 dark:text-zinc-400 mb-3">
              Click on a blueprint to add it to your workflow
            </div>
            <ItemGroup>
              {blueprints.map((block) => {
                const IconComponent = resolveIcon(block.icon || 'git-branch')
                const colorClass = getColorClass(block.color)

                return (
                  <Item
                    key={`${block.type}-${block.name || block.id}`}
                    onClick={() => onBlockClick(block)}
                    className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                    size="sm"
                  >
                    <ItemMedia>
                      <IconComponent size={24} className={colorClass} />
                    </ItemMedia>
                    <ItemContent>
                      <ItemTitle>{block.label || block.name}</ItemTitle>
                      {block.description && (
                        <ItemDescription>{block.description}</ItemDescription>
                      )}
                    </ItemContent>
                  </Item>
                )
              })}
            </ItemGroup>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}
