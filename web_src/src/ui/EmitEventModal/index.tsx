import { useState } from 'react'
import { Play } from 'lucide-react'
import { Dialog, DialogContent, DialogFooter } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Heading } from '@/components/Heading/heading'
import { showSuccessToast, showErrorToast } from '@/utils/toast'
import Editor from '@monaco-editor/react'

interface EmitEventModalProps {
  isOpen: boolean
  onClose: () => void
  nodeId: string
  nodeName: string
  workflowId: string
  organizationId: string
  channels: string[]
  onEmit: (channel: string, data: any) => Promise<void>
}

export const EmitEventModal = ({
  isOpen,
  onClose,
  nodeName,
  channels,
  onEmit,
}: EmitEventModalProps) => {
  const [selectedChannel, setSelectedChannel] = useState<string>(channels[0] || 'default')
  const [eventData, setEventData] = useState<string>('{}')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleClose = () => {
    setSelectedChannel(channels[0] || 'default')
    setEventData('{}')
    setIsSubmitting(false)
    onClose()
  }

  const handleSubmit = async () => {
    try {
      setIsSubmitting(true)

      // Validate JSON
      let parsedData
      try {
        parsedData = JSON.parse(eventData)
      } catch (e) {
        showErrorToast('Invalid JSON format')
        setIsSubmitting(false)
        return
      }

      await onEmit(selectedChannel, parsedData)
      showSuccessToast('Event emitted successfully')
      handleClose()
    } catch (error) {
      console.error('Error emitting event:', error)
      showErrorToast('Failed to emit event')
      setIsSubmitting(false)
    }
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent className="max-w-3xl max-h-[80vh]">
        <div className="space-y-4">
          <div className="flex items-center gap-3">
            <Play className="text-blue-600 dark:text-blue-400" size={24} />
            <Heading level={3} className="!mb-0">Emit Event</Heading>
          </div>

          <div className="text-sm text-zinc-600 dark:text-zinc-400">
            Manually emit an output event for node: <strong>{nodeName}</strong>
          </div>

          <div className="space-y-4">
            <div className="flex items-center gap-3">
              <Label className="min-w-[120px]">Output Channel</Label>
              <Select value={selectedChannel} onValueChange={setSelectedChannel}>
                <SelectTrigger className="flex-1">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {channels.map((channel) => (
                    <SelectItem key={channel} value={channel}>
                      {channel}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="border border-zinc-200 dark:border-zinc-700 rounded-md overflow-hidden">
              <Editor
                height="300px"
                defaultLanguage="json"
                value={eventData}
                onChange={(value) => setEventData(value || '{}')}
                options={{
                  minimap: { enabled: false },
                  fontSize: 13,
                  lineNumbers: 'on',
                  scrollBeyondLastLine: false,
                  automaticLayout: true,
                }}
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={handleClose} disabled={isSubmitting}>
              Cancel
            </Button>
            <Button onClick={handleSubmit} disabled={isSubmitting}>
              <Play size={16} />
              {isSubmitting ? 'Emitting...' : 'Emit Event'}
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  )
}
