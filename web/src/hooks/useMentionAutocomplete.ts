import { useState, useCallback, useRef } from 'react'

interface UseMentionAutocompleteOptions {
  value: string
  onValueChange: (v: string) => void
  textareaRef: React.RefObject<HTMLTextAreaElement | null>
}

/** Compute pixel coordinates of the caret inside a textarea. */
function getCaretCoordinates(textarea: HTMLTextAreaElement, position: number) {
  const cs = getComputedStyle(textarea)
  const mirror = document.createElement('div')
  mirror.style.cssText = `
    position:absolute; visibility:hidden; white-space:pre-wrap; word-wrap:break-word; overflow:hidden;
    width:${cs.width}; font:${cs.font}; letter-spacing:${cs.letterSpacing}; word-spacing:${cs.wordSpacing};
    text-indent:${cs.textIndent}; text-transform:${cs.textTransform}; padding:${cs.padding};
    border-width:${cs.borderTopWidth} ${cs.borderRightWidth} ${cs.borderBottomWidth} ${cs.borderLeftWidth};
    border-style:${cs.borderStyle}; box-sizing:${cs.boxSizing}; line-height:${cs.lineHeight};
    tab-size:${cs.tabSize};
  `
  document.body.appendChild(mirror)

  mirror.textContent = textarea.value.substring(0, position)
  const marker = document.createElement('span')
  marker.textContent = '\u200b' // zero-width space
  mirror.appendChild(marker)

  const top = marker.offsetTop - textarea.scrollTop
  const left = marker.offsetLeft - textarea.scrollLeft
  const height = marker.offsetHeight || parseInt(cs.lineHeight) || parseInt(cs.fontSize) * 1.2

  document.body.removeChild(mirror)
  return { top, left, height }
}

export interface DropdownPosition {
  top: number
  left: number
}

export function useMentionAutocomplete({ value, onValueChange, textareaRef }: UseMentionAutocompleteOptions) {
  const [mentionModalOpen, setMentionModalOpen] = useState(false)
  const [dropdownPosition, setDropdownPosition] = useState<DropdownPosition>({ top: 0, left: 0 })
  const cursorPosRef = useRef(0) // position right after the @

  // Refs to avoid stale closures while dropdown is open
  const valueRef = useRef(value)
  valueRef.current = value
  const onValueChangeRef = useRef(onValueChange)
  onValueChangeRef.current = onValueChange

  const onMentionKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === '@' && !e.ctrlKey && !e.metaKey && !e.altKey) {
        // Let the @ be typed normally — don't preventDefault
        const pos = (e.currentTarget.selectionStart ?? 0) + 1 // after the @
        cursorPosRef.current = pos

        // Calculate position after the character is rendered
        requestAnimationFrame(() => {
          const textarea = textareaRef.current
          if (!textarea) return
          const rect = textarea.getBoundingClientRect()
          const caret = getCaretCoordinates(textarea, pos)
          setDropdownPosition({
            top: rect.top + caret.top + caret.height,
            left: rect.left + caret.left,
          })
          setMentionModalOpen(true)
        })
      }
    },
    [textareaRef],
  )

  const onMentionSelect = useCallback(
    (markdownLink: string) => {
      const pos = cursorPosRef.current // right after the @
      const v = valueRef.current
      const before = v.substring(0, pos - 1) // everything before the @
      const after = v.substring(pos) // everything after the @
      onValueChangeRef.current(before + markdownLink + after)
      setMentionModalOpen(false)

      const newPos = pos - 1 + markdownLink.length
      requestAnimationFrame(() => {
        const el = textareaRef.current
        if (el) {
          el.focus()
          el.setSelectionRange(newPos, newPos)
        }
      })
    },
    [textareaRef],
  )

  const onMentionClose = useCallback(() => {
    // Leave the @ in the textarea, just close the dropdown
    setMentionModalOpen(false)
    requestAnimationFrame(() => {
      const el = textareaRef.current
      if (el) {
        el.focus()
        el.setSelectionRange(cursorPosRef.current, cursorPosRef.current)
      }
    })
  }, [textareaRef])

  return { onMentionKeyDown, mentionModalOpen, dropdownPosition, onMentionSelect, onMentionClose }
}
