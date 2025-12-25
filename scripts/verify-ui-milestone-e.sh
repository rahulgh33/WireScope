#!/bin/bash
# UI Feature Verification Script for Milestone E Demo

set -e

echo "========================================="
echo "Milestone E - UI Feature Verification"
echo "========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0

# Helper function to check if a feature exists
check_feature() {
    local feature=$1
    local file=$2
    local pattern=$3
    
    echo -n "  Checking $feature... "
    if [ -f "$file" ] && grep -q "$pattern" "$file"; then
        echo -e "${GREEN}✓${NC}"
        PASSED=$((PASSED + 1))
        return 0
    else
        echo -e "${RED}✗${NC}"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

cd "$(dirname "$0")/.."

echo "1. Core UI Architecture"
echo "======================="
check_feature "React Router setup" "web/src/App.tsx" "BrowserRouter"
check_feature "React Query client" "web/src/App.tsx" "QueryClient"
check_feature "Layout component" "web/src/components/layout/Layout.tsx" "export.*Layout"
check_feature "API client" "web/src/lib/api.ts" "class APIClient"
echo ""

echo "2. Authentication"
echo "================="
check_feature "Login page" "web/src/features/auth/LoginPage.tsx" "LoginPage"
check_feature "Auth store" "web/src/stores/authStore.ts" "useAuthStore"
check_feature "Protected routes" "web/src/features/auth/ProtectedRoute.tsx" "ProtectedRoute"
echo ""

echo "3. Dashboard (Task 32)"
echo "======================"
check_feature "Dashboard page" "web/src/features/dashboard/DashboardPage.tsx" "DashboardPage"
check_feature "Metric cards" "web/src/components/dashboard/MetricCards.tsx" "MetricCards"
check_feature "Latency chart" "web/src/components/dashboard/LatencyTrendsChart.tsx" "LatencyTrendsChart"
check_feature "Top issues panel" "web/src/components/dashboard/TopIssuesPanel.tsx" "TopIssuesPanel"
check_feature "Diagnostics panel" "web/src/components/dashboard/RecentDiagnosticsPanel.tsx" "RecentDiagnosticsPanel"
echo ""

echo "4. Client Management (Task 33)"
echo "==============================="
check_feature "Clients page" "web/src/features/clients/ClientsPage.tsx" "ClientsPage"
check_feature "Client card" "web/src/components/clients/ClientCard.tsx" "ClientCard"
check_feature "Client detail page" "web/src/features/clients/ClientDetailPage.tsx" "ClientDetailPage"
check_feature "Latency breakdown" "web/src/components/clients/LatencyBreakdownChart.tsx" "LatencyBreakdownChart"
check_feature "Per-target performance" "web/src/components/clients/PerTargetPerformance.tsx" "PerTargetPerformance"
echo ""

echo "5. Target Management (Task 33)"
echo "==============================="
check_feature "Targets page" "web/src/features/targets/TargetsPage.tsx" "TargetsPage"
check_feature "Target card" "web/src/components/targets/TargetCard.tsx" "TargetCard"
check_feature "Target detail page" "web/src/features/targets/TargetDetailPage.tsx" "TargetDetailPage"
check_feature "Per-client performance" "web/src/components/targets/PerClientPerformance.tsx" "PerClientPerformance"
check_feature "Common issues" "web/src/components/targets/CommonIssuesPanel.tsx" "CommonIssuesPanel"
echo ""

echo "6. Real-Time Monitoring (Task 35)"
echo "=================================="
check_feature "Live updates hook" "web/src/hooks/useLiveUpdates.ts" "useLiveUpdates"
check_feature "Live dashboard" "web/src/components/dashboard/LiveDashboard.tsx" "LiveDashboard"
check_feature "Live event stream" "web/src/components/realtime/LiveEventStream.tsx" "LiveEventStream"
check_feature "Live probe status" "web/src/components/realtime/LiveProbeStatus.tsx" "LiveProbeStatus"
check_feature "Connection status" "web/src/components/ui/ConnectionStatus.tsx" "ConnectionStatus"
check_feature "Notification system" "web/src/components/realtime/NotificationSystem.tsx" "NotificationSystem"
echo ""

echo "7. Admin & Configuration (Task 36)"
echo "==================================="
check_feature "Admin page" "web/src/pages/AdminPage.tsx" "AdminPage"
check_feature "System health" "web/src/pages/AdminPage.tsx" "SystemHealthDashboard"
check_feature "Probe management" "web/src/pages/AdminPage.tsx" "ProbeManagement"
check_feature "Token management" "web/src/pages/AdminPage.tsx" "TokenManagement"
check_feature "User management" "web/src/pages/AdminPage.tsx" "UserManagement"
check_feature "Database maintenance" "web/src/pages/AdminPage.tsx" "DatabaseManagement"
echo ""

echo "8. UI Polish & Accessibility (Task 37)"
echo "======================================="
check_feature "Error handling" "web/src/components/ui/ErrorHandling.tsx" "ErrorBoundary"
check_feature "Loading states" "web/src/components/ui/ErrorHandling.tsx" "LoadingSpinner"
check_feature "Accessibility utils" "web/src/components/ui/Accessibility.tsx" "Tooltip"
check_feature "Dark mode (CSS)" "web/src/index.css" "data-theme=\"dark\""
check_feature "Dark mode (store)" "web/src/stores/uiStore.ts" "theme"
check_feature "Responsive design" "web/tailwind.config.js" "tailwindcss"
echo ""

echo "9. UI Components"
echo "================"
check_feature "Button component" "web/src/components/ui/Button.tsx" "Button"
check_feature "Card component" "web/src/components/ui/Card.tsx" "Card"
check_feature "Tabs component" "web/src/components/ui/tabs.tsx" "Tabs"
check_feature "Badge component" "web/src/components/ui/badge.tsx" "Badge"
check_feature "Skeleton loader" "web/src/components/ui/Skeleton.tsx" "Skeleton"
check_feature "Scroll area" "web/src/components/ui/scroll-area.tsx" "ScrollArea"
echo ""

echo "10. Hooks & State Management"
echo "============================"
check_feature "useMetrics hook" "web/src/hooks/useMetrics.ts" "useMetrics"
check_feature "useClients hook" "web/src/hooks/useClients.ts" "useClients"
check_feature "useTargets hook" "web/src/hooks/useTargets.ts" "useTargets"
check_feature "Auth store" "web/src/stores/authStore.ts" "useAuthStore"
check_feature "UI store" "web/src/stores/uiStore.ts" "useUIStore"
echo ""

echo "11. Type Definitions"
echo "===================="
check_feature "API types" "web/src/types/api.ts" "DashboardOverview"
check_feature "Client types" "web/src/types/api.ts" "Client"
check_feature "Target types" "web/src/types/api.ts" "Target"
check_feature "Diagnostic types" "web/src/types/api.ts" "Diagnostic"
echo ""

echo "12. Documentation"
echo "================="
check_feature "UI architecture" "docs/UI_ARCHITECTURE.md" "UI Architecture"
check_feature "Testing guide" "docs/TESTING_GUIDE.md" "Testing Guide"
check_feature "Accessibility guide" "docs/ACCESSIBILITY_GUIDE.md" "Accessibility Guide"
check_feature "Milestone E demo" "docs/MILESTONE_E_DEMO.md" "Milestone E Demo"
echo ""

echo "13. Build Verification"
echo "======================"
echo -n "  Checking package.json... "
if [ -f "web/package.json" ] && grep -q "\"build\":" "web/package.json"; then
    echo -e "${GREEN}✓${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗${NC}"
    FAILED=$((FAILED + 1))
fi

echo -n "  Checking tsconfig... "
if [ -f "web/tsconfig.json" ]; then
    echo -e "${GREEN}✓${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗${NC}"
    FAILED=$((FAILED + 1))
fi

echo -n "  Checking vite config... "
if [ -f "web/vite.config.ts" ]; then
    echo -e "${GREEN}✓${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗${NC}"
    FAILED=$((FAILED + 1))
fi

echo -n "  Building UI... "
cd web
if npm run build > /tmp/ui-build.log 2>&1; then
    echo -e "${GREEN}✓${NC}"
    PASSED=$((PASSED + 1))
    echo "    Build output: $(du -sh dist 2>/dev/null || echo 'N/A')"
else
    echo -e "${RED}✗${NC}"
    FAILED=$((FAILED + 1))
    echo "    See /tmp/ui-build.log for details"
fi
cd ..
echo ""

echo "========================================="
echo "Summary"
echo "========================================="
echo ""
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✨ All features verified! Milestone E is complete!${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠ Some features are missing or incomplete${NC}"
    exit 1
fi
