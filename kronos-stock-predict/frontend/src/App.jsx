import React, { useState, useEffect } from 'react'
import StockList from './components/StockList'
import StockDetail from './components/StockDetail'
import DataManager from './components/DataManager'

function App() {
  const [view, setView] = useState('list')
  const [selectedStock, setSelectedStock] = useState(null)
  const [stocks, setStocks] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    fetchStocks()
  }, [])

  const fetchStocks = async () => {
    try {
      setLoading(true)
      const response = await fetch('/api/stocks')
      if (!response.ok) throw new Error('Failed to fetch stocks')
      const data = await response.json()
      setStocks(data)
      setError(null)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleSelectStock = (stock) => {
    setSelectedStock(stock)
    setView('detail')
  }

  const handleBack = () => {
    setView('list')
    setSelectedStock(null)
  }

  const handleRefresh = () => {
    fetchStocks()
  }

  if (loading && stocks.length === 0) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto"></div>
          <p className="mt-4 text-gray-400">加载中...</p>
        </div>
      </div>
    )
  }

  if (error && stocks.length === 0) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <p className="text-red-400 mb-4">{error}</p>
          <button
            onClick={fetchStocks}
            className="px-4 py-2 bg-blue-600 rounded hover:bg-blue-700"
          >
            重试
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen">
      <header className="bg-slate-800 border-b border-slate-700 px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            {view !== 'list' && (
              <button
                onClick={handleBack}
                className="px-3 py-1.5 bg-slate-700 rounded hover:bg-slate-600 text-sm"
              >
                返回
              </button>
            )}
            <h1 className="text-xl font-bold text-white">Kronos 股票预测系统</h1>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={handleRefresh}
              className="px-3 py-1.5 bg-slate-700 rounded hover:bg-slate-600 text-sm"
            >
              刷新数据
            </button>
            <button
              onClick={() => setView(view === 'manager' ? 'list' : 'manager')}
              className="px-3 py-1.5 bg-blue-600 rounded hover:bg-blue-700 text-sm"
            >
              数据管理
            </button>
          </div>
        </div>
      </header>

      <main className="p-6">
        {view === 'list' && (
          <StockList
            stocks={stocks}
            onSelectStock={handleSelectStock}
          />
        )}
        {view === 'detail' && selectedStock && (
          <StockDetail
            stock={selectedStock}
            onBack={handleBack}
          />
        )}
        {view === 'manager' && (
          <DataManager
            stocks={stocks}
            onBack={() => setView('list')}
          />
        )}
      </main>
    </div>
  )
}

export default App
