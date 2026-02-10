package server

import "html/template"

// swaggerThemeCSS returns HTML to inject into Swagger UI document head for Wintry-like dark theme.
func swaggerThemeCSS() template.HTML {
	const css = `<style>
	/* Wintry-style dark theme for Swagger UI at /docs */
	body { background: #1a1d21 !important; color: #e4e6eb !important; }
	.swagger-ui { font-family: system-ui, -apple-system, sans-serif !important; }
	.swagger-ui .topbar { background: #25282c !important; border-bottom: 1px solid #2d3139 !important; }
	.swagger-ui .topbar .link { color: #e4e6eb !important; }
	.swagger-ui .info .title { color: #e4e6eb !important; }
	.swagger-ui .info p, .swagger-ui .info table { color: #b0b4bb !important; }
	.swagger-ui .info a { color: #58a6ff !important; }
	.swagger-ui .opblock { border: 1px solid #2d3139 !important; background: #25282c !important; margin: 0 0 1px 0 !important; }
	.swagger-ui .opblock .opblock-summary-method { background: #2d3139 !important; color: #e4e6eb !important; }
	.swagger-ui .opblock.opblock-get .opblock-summary-method { background: #1391ff !important; }
	.swagger-ui .opblock.opblock-post .opblock-summary-method { background: #49cc90 !important; }
	.swagger-ui .opblock.opblock-put .opblock-summary-method { background: #fca130 !important; }
	.swagger-ui .opblock.opblock-delete .opblock-summary-method { background: #f93e3e !important; }
	.swagger-ui .opblock .opblock-summary-path, .swagger-ui .opblock .opblock-summary-description { color: #e4e6eb !important; }
	.swagger-ui .opblock .opblock-section-header { background: #25282c !important; border: 1px solid #2d3139 !important; color: #e4e6eb !important; }
	.swagger-ui .opblock .opblock-section-header label { color: #e4e6eb !important; }
	.swagger-ui .table-container { background: #1a1d21 !important; border: 1px solid #2d3139 !important; }
	.swagger-ui table thead td { border-bottom: 1px solid #2d3139 !important; color: #b0b4bb !important; background: #25282c !important; }
	.swagger-ui table tbody td { border-bottom: 1px solid #2d3139 !important; color: #e4e6eb !important; }
	.swagger-ui .parameter__name { color: #e4e6eb !important; }
	.swagger-ui .parameter__type { color: #58a6ff !important; }
	.swagger-ui input[type=text], .swagger-ui textarea, .swagger-ui select { background: #1a1d21 !important; border: 1px solid #2d3139 !important; color: #e4e6eb !important; }
	.swagger-ui .btn { background: #25282c !important; border: 1px solid #2d3139 !important; color: #58a6ff !important; }
	.swagger-ui .btn.execute { background: #1391ff !important; border-color: #1391ff !important; color: #fff !important; }
	.swagger-ui .btn.authorize { border-color: #49cc90 !important; color: #49cc90 !important; }
	.swagger-ui .btn.authorize svg { fill: #49cc90 !important; }
	.swagger-ui .responses-inner h4, .swagger-ui .responses-inner h5 { color: #e4e6eb !important; }
	.swagger-ui .response-col_status { color: #b0b4bb !important; }
	.swagger-ui .response-col_description { color: #e4e6eb !important; }
	.swagger-ui .model-box { background: #25282c !important; border: 1px solid #2d3139 !important; }
	.swagger-ui .model-box-control { color: #58a6ff !important; }
	.swagger-ui .model .property { color: #e4e6eb !important; }
	.swagger-ui .model .prop-type { color: #58a6ff !important; }
	.swagger-ui .scheme-container { background: #25282c !important; border: 1px solid #2d3139 !important; box-shadow: none !important; }
	.swagger-ui .scheme-container .scheme-info { color: #e4e6eb !important; }
	.swagger-ui .loading-container .loading { border-color: #2d3139 !important; border-top-color: #58a6ff !important; }
	.swagger-ui select { color: #e4e6eb !important; }
	.swagger-ui .opblock-tag { border-bottom: 1px solid #2d3139 !important; }
	.swagger-ui .opblock-tag .opblock-tag-section { color: #e4e6eb !important; }
	.swagger-ui .opblock-tag .opblock-tag-section:hover { background: #25282c !important; }
	.swagger-ui .tab li { color: #b0b4bb !important; }
	.swagger-ui .tab li.active { color: #58a6ff !important; }
	.swagger-ui .tab li button { color: inherit !important; }
	.swagger-ui .dialog-ux .modal-ux { background: #25282c !important; border: 1px solid #2d3139 !important; }
	.swagger-ui .dialog-ux .modal-ux-header { border-bottom: 1px solid #2d3139 !important; color: #e4e6eb !important; }
	.swagger-ui .dialog-ux .modal-ux-content { color: #e4e6eb !important; }
	</style>`
	return template.HTML(css)
}
