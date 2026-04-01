<template>
  <div class="flex h-screen bg-background text-foreground overflow-hidden">
    <!-- Sidebar -->
    <aside
      class="border-r bg-muted/20 transition-all duration-300 flex flex-col z-20"
      :class="isCollapse ? 'w-16' : 'w-64'"
    >
      <div class="h-14 flex items-center px-4 border-b bg-background" :class="isCollapse ? 'justify-center' : 'justify-between'">
        <div class="flex items-center gap-2 font-semibold text-lg tracking-tight">
          <div class="w-8 h-8 rounded-md bg-primary text-primary-foreground flex items-center justify-center font-bold">
            B
          </div>
          <span v-if="!isCollapse">BigOps</span>
        </div>
      </div>

      <ScrollArea class="flex-1 py-4">
        <nav class="grid gap-1 px-2">
          <template v-for="menu in menuTree" :key="menu.id">
            <!-- 带有子菜单的项 -->
            <Collapsible v-if="menu.children?.length && menu.type !== 3" class="w-full">
              <CollapsibleTrigger as-child>
                <Button variant="ghost" class="w-full justify-between font-normal mb-1" :class="{ 'px-2': isCollapse }">
                  <div class="flex items-center gap-3">
                    <el-icon class="text-lg"><component :is="menu.icon || 'Folder'" /></el-icon>
                    <span v-if="!isCollapse">{{ menu.title }}</span>
                  </div>
                  <el-icon v-if="!isCollapse" class="text-xs transition-transform duration-200"><ArrowDown /></el-icon>
                </Button>
              </CollapsibleTrigger>
              <CollapsibleContent v-if="!isCollapse" class="pl-4 space-y-1 mt-1">
                <template v-for="child in menu.children" :key="child.id">
                  <Button
                    v-if="child.type !== 3 && child.path"
                    variant="ghost"
                    size="sm"
                    class="w-full justify-start font-normal text-muted-foreground hover:text-foreground"
                    :class="{ 'bg-secondary text-foreground font-medium': route.path === child.path }"
                    @click="router.push(child.path)"
                  >
                    <el-icon class="mr-2 text-base"><component :is="child.icon || 'Document'" /></el-icon>
                    {{ child.title }}
                  </Button>
                </template>
              </CollapsibleContent>
            </Collapsible>

            <!-- 无子菜单的单项 -->
            <Button
              v-else-if="menu.type !== 3 && menu.path"
              variant="ghost"
              class="w-full justify-start font-normal mb-1"
              :class="{ 'bg-secondary text-foreground font-medium': route.path === menu.path, 'px-2': isCollapse }"
              @click="router.push(menu.path)"
            >
              <el-icon class="mr-3 text-lg"><component :is="menu.icon || 'Document'" /></el-icon>
              <span v-if="!isCollapse">{{ menu.title }}</span>
            </Button>
          </template>
        </nav>
      </ScrollArea>
    </aside>

    <!-- Main Content -->
    <div class="flex flex-col flex-1 overflow-hidden relative">
      <!-- Topbar -->
      <header class="h-14 border-b bg-background flex items-center justify-between px-4 z-10 shrink-0">
        <div class="flex items-center gap-4">
          <Button variant="ghost" size="icon" class="h-8 w-8 text-muted-foreground" @click="isCollapse = !isCollapse">
            <el-icon class="text-xl"><Fold v-if="!isCollapse" /><Expand v-else /></el-icon>
          </Button>

          <Breadcrumb class="hidden sm:flex">
            <BreadcrumbList>
              <BreadcrumbItem>
                <BreadcrumbLink href="/dashboard">首页</BreadcrumbLink>
              </BreadcrumbItem>
              <template v-for="(item, index) in route.matched.filter(r => r.meta?.title)" :key="item.path">
                <BreadcrumbSeparator />
                <BreadcrumbItem>
                  <BreadcrumbPage v-if="index === route.matched.filter(r => r.meta?.title).length - 1">{{ item.meta.title }}</BreadcrumbPage>
                  <BreadcrumbLink v-else :href="item.path">{{ item.meta.title }}</BreadcrumbLink>
                </BreadcrumbItem>
              </template>
            </BreadcrumbList>
          </Breadcrumb>
        </div>

        <div class="flex items-center gap-2">
          <!-- Cmd+K Search Trigger -->
          <Button variant="outline" class="hidden sm:flex items-center gap-2 text-muted-foreground h-8 px-3" @click="cmdPaletteVisible = true">
            <el-icon><Search /></el-icon>
            <span class="text-xs font-normal">Search...</span>
            <kbd class="pointer-events-none inline-flex h-5 select-none items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground opacity-100"><span class="text-xs">⌘</span>K</kbd>
          </Button>

          <!-- Notifications -->
          <div class="relative flex items-center justify-center">
            <Button variant="ghost" size="icon" class="h-8 w-8 relative text-muted-foreground" @click="openNotifications">
              <el-icon class="text-lg"><Bell /></el-icon>
              <span v-if="unreadCount > 0" class="absolute top-1 right-1 w-2 h-2 rounded-full bg-destructive border-2 border-background"></span>
            </Button>
          </div>

          <!-- User Menu -->
          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" class="h-8 pl-2 pr-1 ml-2 flex items-center gap-2 border border-transparent hover:border-border hover:bg-secondary">
                <Avatar class="h-6 w-6">
                  <AvatarFallback class="bg-primary/10 text-primary text-xs">{{ userStore.userInfo?.username?.charAt(0).toUpperCase() }}</AvatarFallback>
                </Avatar>
                <span class="text-sm font-medium hidden sm:block">{{ userStore.userInfo?.username }}</span>
                <el-icon class="text-muted-foreground"><ArrowDown /></el-icon>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" class="w-48">
              <DropdownMenuLabel>My Account</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem @click="router.push('/user/settings')">
                <el-icon class="mr-2"><UserFilled /></el-icon>
                <span>Profile Settings</span>
              </DropdownMenuItem>
              <DropdownMenuItem @click="router.push('/notification/console')">
                <el-icon class="mr-2"><Setting /></el-icon>
                <span>System Config</span>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem @click="handleLogout" class="text-destructive focus:bg-destructive focus:text-destructive-foreground">
                <el-icon class="mr-2"><SwitchButton /></el-icon>
                <span>Log out</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </header>

      <!-- Tags Bar -->
      <div 
        class="h-10 bg-muted/30 border-b flex items-center px-4 overflow-x-auto whitespace-nowrap scrollbar-hide shrink-0"
        ref="tagsBarRef"
        @wheel.prevent="handleTagsScroll"
      >
        <div class="flex gap-1.5 w-max min-w-full">
          <div
            v-for="tag in tagsStore.visitedViews"
            :key="tag.path"
            class="group flex-shrink-0 flex items-center gap-1.5 px-3 py-1 text-xs border rounded-full cursor-pointer transition-colors bg-background"
            :class="tag.path === route.path ? 'border-primary text-primary font-medium' : 'border-border text-muted-foreground hover:bg-muted'"
            @click="router.push(tag.path)"
            @contextmenu="onTagContextMenu(tag.path, $event)"
          >
            <span>{{ tag.title }}</span>
            <div 
              v-if="tag.closable" 
              class="w-4 h-4 flex items-center justify-center rounded-full hover:bg-muted-foreground/20 transition-colors"
              @click.stop="handleTabRemove(tag.path)"
            >
              <el-icon class="t
