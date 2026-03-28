import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useViewStateStore = defineStore('viewState', () => {
  const ticketListDirty = ref(false)
  const ticketTypeVersion = ref(0)
  const requestTemplateVersion = ref(0)

  function markTicketListDirty() {
    ticketListDirty.value = true
  }

  function consumeTicketListDirty(): boolean {
    const dirty = ticketListDirty.value
    ticketListDirty.value = false
    return dirty
  }

  function markTicketTypeDirty() {
    ticketTypeVersion.value += 1
  }

  function markRequestTemplateDirty() {
    requestTemplateVersion.value += 1
  }

  return {
    ticketListDirty,
    ticketTypeVersion,
    requestTemplateVersion,
    markTicketListDirty,
    consumeTicketListDirty,
    markTicketTypeDirty,
    markRequestTemplateDirty,
  }
})
