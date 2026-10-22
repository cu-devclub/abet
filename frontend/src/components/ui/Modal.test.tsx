import { useState } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Modal } from './Modal'

function Harness({ onClose }: { onClose: () => void }) {
  const [open, setOpen] = useState(true)
  return (
    <>
      <button type="button">outside</button>
      <Modal
        isOpen={open}
        onClose={() => {
          onClose()
          setOpen(false)
        }}
        title="Edit thing"
        footer={<button type="button">Save</button>}
      >
        <input aria-label="Name" />
      </Modal>
    </>
  )
}

describe('Modal', () => {
  it('is an accessible dialog labelled by its title', () => {
    render(<Harness onClose={() => undefined} />)
    expect(screen.getByRole('dialog', { name: 'Edit thing' })).toBeInTheDocument()
  })

  it('closes on Escape', async () => {
    const onClose = vi.fn()
    render(<Harness onClose={onClose} />)
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('keeps Tab focus inside the dialog', async () => {
    const user = userEvent.setup()
    render(<Harness onClose={() => undefined} />)
    const dialog = screen.getByRole('dialog')
    for (let i = 0; i < 6; i++) {
      await user.tab()
      expect(dialog).toContainElement(document.activeElement as HTMLElement)
    }
    await user.tab({ shift: true })
    expect(dialog).toContainElement(document.activeElement as HTMLElement)
  })

  it('returns focus to the opener when closed', async () => {
    const user = userEvent.setup()
    function Opener() {
      const [open, setOpen] = useState(false)
      return (
        <>
          <button type="button" onClick={() => setOpen(true)}>
            open
          </button>
          <Modal isOpen={open} onClose={() => setOpen(false)} title="T">
            <p>body</p>
          </Modal>
        </>
      )
    }
    render(<Opener />)
    const opener = screen.getByRole('button', { name: 'open' })
    await user.click(opener)
    await user.keyboard('{Escape}')
    expect(opener).toHaveFocus()
  })
})
