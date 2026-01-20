// Lightweight charting library for Status Incident
// No external dependencies - pure SVG rendering

const Charts = {
    // Create a line chart for latency data
    createLatencyChart: function(containerId, data, options = {}) {
        const container = document.getElementById(containerId);
        if (!container || !data || data.length === 0) {
            if (container) container.innerHTML = '<p class="chart-empty">No latency data available</p>';
            return;
        }

        const width = options.width || container.clientWidth || 600;
        const height = options.height || 200;
        const padding = { top: 20, right: 20, bottom: 30, left: 50 };
        const chartWidth = width - padding.left - padding.right;
        const chartHeight = height - padding.top - padding.bottom;

        // Find min/max values
        const values = data.map(d => d.avg_ms);
        const minVal = Math.min(...values) * 0.9;
        const maxVal = Math.max(...values) * 1.1;
        const range = maxVal - minVal || 1;

        // Scale functions
        const scaleX = (i) => padding.left + (i / (data.length - 1 || 1)) * chartWidth;
        const scaleY = (v) => padding.top + chartHeight - ((v - minVal) / range) * chartHeight;

        // Build SVG
        let svg = `<svg width="${width}" height="${height}" class="latency-chart">`;

        // Grid lines
        svg += '<g class="grid">';
        for (let i = 0; i <= 4; i++) {
            const y = padding.top + (chartHeight / 4) * i;
            const val = maxVal - (range / 4) * i;
            svg += `<line x1="${padding.left}" y1="${y}" x2="${width - padding.right}" y2="${y}" stroke="#e5e7eb" stroke-dasharray="3,3"/>`;
            svg += `<text x="${padding.left - 5}" y="${y + 4}" text-anchor="end" class="axis-label">${Math.round(val)}ms</text>`;
        }
        svg += '</g>';

        // Area fill
        let areaPath = `M${scaleX(0)},${padding.top + chartHeight}`;
        data.forEach((d, i) => {
            areaPath += ` L${scaleX(i)},${scaleY(d.avg_ms)}`;
        });
        areaPath += ` L${scaleX(data.length - 1)},${padding.top + chartHeight} Z`;
        svg += `<path d="${areaPath}" fill="url(#latencyGradient)" opacity="0.3"/>`;

        // Gradient definition
        svg += `<defs>
            <linearGradient id="latencyGradient" x1="0%" y1="0%" x2="0%" y2="100%">
                <stop offset="0%" style="stop-color:#3b82f6;stop-opacity:0.8"/>
                <stop offset="100%" style="stop-color:#3b82f6;stop-opacity:0"/>
            </linearGradient>
        </defs>`;

        // Line path
        let linePath = `M${scaleX(0)},${scaleY(data[0].avg_ms)}`;
        data.forEach((d, i) => {
            if (i > 0) linePath += ` L${scaleX(i)},${scaleY(d.avg_ms)}`;
        });
        svg += `<path d="${linePath}" fill="none" stroke="#3b82f6" stroke-width="2"/>`;

        // Data points
        svg += '<g class="points">';
        data.forEach((d, i) => {
            const x = scaleX(i);
            const y = scaleY(d.avg_ms);
            const color = d.failures > 0 ? '#ef4444' : '#3b82f6';
            svg += `<circle cx="${x}" cy="${y}" r="4" fill="${color}" class="data-point" data-value="${Math.round(d.avg_ms)}ms" data-time="${d.timestamp}"/>`;
        });
        svg += '</g>';

        // X-axis labels (show a few)
        svg += '<g class="x-axis">';
        const labelCount = Math.min(6, data.length);
        for (let i = 0; i < labelCount; i++) {
            const idx = Math.floor(i * (data.length - 1) / (labelCount - 1 || 1));
            const x = scaleX(idx);
            const time = new Date(data[idx].timestamp);
            const label = time.getHours().toString().padStart(2, '0') + ':' + time.getMinutes().toString().padStart(2, '0');
            svg += `<text x="${x}" y="${height - 5}" text-anchor="middle" class="axis-label">${label}</text>`;
        }
        svg += '</g>';

        svg += '</svg>';
        container.innerHTML = svg;

        // Add tooltip handlers
        container.querySelectorAll('.data-point').forEach(point => {
            point.addEventListener('mouseenter', (e) => {
                const tooltip = document.createElement('div');
                tooltip.className = 'chart-tooltip';
                tooltip.innerHTML = `${e.target.dataset.value}<br><small>${new Date(e.target.dataset.time).toLocaleString()}</small>`;
                tooltip.style.left = e.pageX + 'px';
                tooltip.style.top = (e.pageY - 40) + 'px';
                document.body.appendChild(tooltip);
            });
            point.addEventListener('mouseleave', () => {
                document.querySelectorAll('.chart-tooltip').forEach(t => t.remove());
            });
        });
    },

    // Create an uptime heatmap (GitHub-style)
    createUptimeHeatmap: function(containerId, data, options = {}) {
        const container = document.getElementById(containerId);
        if (!container || !data || data.length === 0) {
            if (container) container.innerHTML = '<p class="chart-empty">No uptime data available</p>';
            return;
        }

        const cellSize = options.cellSize || 12;
        const cellGap = options.cellGap || 2;
        const weeksToShow = Math.ceil(data.length / 7);
        const width = weeksToShow * (cellSize + cellGap) + 40;
        const height = 7 * (cellSize + cellGap) + 30;

        // Color scale based on uptime
        const getColor = (uptime) => {
            if (uptime >= 99.9) return '#22c55e';  // green-500
            if (uptime >= 99) return '#86efac';    // green-300
            if (uptime >= 95) return '#fde047';    // yellow-300
            if (uptime >= 90) return '#fb923c';    // orange-400
            return '#ef4444';                       // red-500
        };

        let svg = `<svg width="${width}" height="${height}" class="uptime-heatmap">`;

        // Day labels
        const days = ['Mon', '', 'Wed', '', 'Fri', '', 'Sun'];
        days.forEach((day, i) => {
            if (day) {
                svg += `<text x="0" y="${20 + i * (cellSize + cellGap) + cellSize/2 + 4}" class="axis-label">${day}</text>`;
            }
        });

        // Cells
        data.forEach((d, i) => {
            const date = new Date(d.date);
            const dayOfWeek = (date.getDay() + 6) % 7; // Monday = 0
            const week = Math.floor(i / 7);
            const x = 35 + week * (cellSize + cellGap);
            const y = 15 + dayOfWeek * (cellSize + cellGap);
            const color = d.total_checks > 0 ? getColor(d.uptime_percent) : '#f3f4f6';

            svg += `<rect x="${x}" y="${y}" width="${cellSize}" height="${cellSize}" rx="2" fill="${color}" class="heatmap-cell" data-date="${d.date}" data-uptime="${d.uptime_percent.toFixed(2)}%" data-checks="${d.total_checks}"/>`;
        });

        svg += '</svg>';

        // Legend
        svg += `<div class="heatmap-legend">
            <span>Less</span>
            <span class="legend-box" style="background:#ef4444"></span>
            <span class="legend-box" style="background:#fb923c"></span>
            <span class="legend-box" style="background:#fde047"></span>
            <span class="legend-box" style="background:#86efac"></span>
            <span class="legend-box" style="background:#22c55e"></span>
            <span>More</span>
        </div>`;

        container.innerHTML = svg;

        // Add tooltip handlers
        container.querySelectorAll('.heatmap-cell').forEach(cell => {
            cell.addEventListener('mouseenter', (e) => {
                const tooltip = document.createElement('div');
                tooltip.className = 'chart-tooltip';
                tooltip.innerHTML = `<strong>${e.target.dataset.date}</strong><br>Uptime: ${e.target.dataset.uptime}<br>Checks: ${e.target.dataset.checks}`;
                tooltip.style.left = e.pageX + 'px';
                tooltip.style.top = (e.pageY - 50) + 'px';
                document.body.appendChild(tooltip);
            });
            cell.addEventListener('mouseleave', () => {
                document.querySelectorAll('.chart-tooltip').forEach(t => t.remove());
            });
        });
    },

    // Load and render charts for a dependency
    loadDependencyCharts: function(dependencyId, containerId) {
        const latencyContainer = document.getElementById(containerId + '-latency');
        const uptimeContainer = document.getElementById(containerId + '-uptime');

        // Load latency data
        if (latencyContainer) {
            fetch(`/api/dependencies/${dependencyId}/latency?period=24h`)
                .then(r => r.json())
                .then(data => {
                    if (data.data_points && data.data_points.length > 0) {
                        this.createLatencyChart(containerId + '-latency', data.data_points);
                    } else {
                        latencyContainer.innerHTML = '<p class="chart-empty">No latency data yet. Data will appear after heartbeat checks.</p>';
                    }
                })
                .catch(err => {
                    latencyContainer.innerHTML = '<p class="chart-empty">Failed to load latency data</p>';
                });
        }

        // Load uptime heatmap
        if (uptimeContainer) {
            fetch(`/api/dependencies/${dependencyId}/uptime?days=90`)
                .then(r => r.json())
                .then(data => {
                    if (data && data.length > 0) {
                        this.createUptimeHeatmap(containerId + '-uptime', data);
                    } else {
                        uptimeContainer.innerHTML = '<p class="chart-empty">No uptime data yet.</p>';
                    }
                })
                .catch(err => {
                    uptimeContainer.innerHTML = '<p class="chart-empty">Failed to load uptime data</p>';
                });
        }
    }
};
