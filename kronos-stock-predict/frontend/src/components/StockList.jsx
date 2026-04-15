import React, { useState, useEffect, useMemo } from 'react'

function StockList({ stocks, onSelectStock }) {
  const [search, setSearch] = useState('')
  const [sortField, setSortField] = useState('code')
  const [sortDir, setSortDir] = useState('asc')
  const [page, setPage] = useState(1)
  const [predictions, setPredictions] = useState({})
  const [loading, setLoading] = useState(true)
  const pageSize = 50

  useEffect(() => {
    fetchPredictions()
  }, [])

  const fetchPredictions = async () => {
    try {
      const response = await fetch('/api/predictions')
      if (response.ok) {
        const data = await response.json()
        const predMap = {}
        data.forEach(sp => {
          predMap[sp.stock.code] = sp
        })
        setPredictions(predMap)
      }
    } catch (err) {
      console.error('Failed to fetch predictions:', err)
    } finally {
      setLoading(false)
    }
  }

  const filteredStocks = useMemo(() => {
    if (!search) return stocks
    const lower = search.toLowerCase()
    return stocks.filter(
      s => s.code.toLowerCase().includes(lower) || s.name.toLowerCase().includes(lower)
    )
  }, [stocks, search])

  const sortedStocks = useMemo(() => {
    return [...filteredStocks].sort((a, b) => {
      let aVal = a[sortField]
      let bVal = b[sortField]
      if (typeof aVal === 'string') {
        aVal = aVal.toLowerCase()
        bVal = bVal.toLowerCase()
      }
      if (sortDir === 'asc') {
        return aVal < bVal ? -1 : aVal > bVal ? 1 : 0
      } else {
        return aVal > bVal ? -1 : aVal < bVal ? 1 : 0
      }
    })
  }, [filteredStocks, sortField, sortDir])

  const totalPages = Math.ceil(sortedStocks.length / pageSize)
  const paginatedStocks = sortedStocks.slice((page - 1) * pageSize, page * pageSize)

  const handleSort = (field) => {
    if (field === sortField) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDir('asc')
    }
  }

  const SortIcon = ({ field }) => {
    if (field !== sortField) return <span className="text-gray-500 ml-1">↕</span>
    return <span className="text-blue-400 ml-1">{sortDir === 'asc' ? '↑' : '↓'}</span>
  }

  const getScore = (code) => {
    const pred = predictions[code]
    if (!pred || !pred.predictions || pred.predictions.length === 0) return null
    return pred.predictions[0]
  }

  const getScoreColor = (score) => {
    if (!score) return 'text-gray-400'
    if (score >= 0.7) return 'text-green-400'
    if (score >= 0.5) return 'text-yellow-400'
    return 'text-red-400'
  }

  return (
    <div>
      <div className="mb-4 flex items-center gap-4">
        <input
          type="text"
          placeholder="搜索股票代码或名称..."
          value={search}
          onChange={e => { setSearch(e.target.value); setPage(1) }}
          className="px-4 py-2 bg-slate-800 border border-slate-600 rounded w-64 text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
        />
        <span className="text-gray-400 text-sm">
          共 {filteredStocks.length} 只股票
        </span>
        {loading && <span className="text-blue-400 text-sm">加载预测数据...</span>}
      </div>

      <div className="bg-slate-800 rounded-lg overflow-hidden">
        <table className="w-full">
          <thead className="bg-slate-700">
            <tr>
              <th
                className="px-4 py-3 text-left text-sm font-medium text-gray-300 cursor-pointer hover:text-white"
                onClick={() => handleSort('code')}
              >
                代码 <SortIcon field="code" />
              </th>
              <th
                className="px-4 py-3 text-left text-sm font-medium text-gray-300 cursor-pointer hover:text-white"
                onClick={() => handleSort('name')}
              >
                名称 <SortIcon field="name" />
              </th>
              <th
                className="px-4 py-3 text-right text-sm font-medium text-gray-300 cursor-pointer hover:text-white"
                onClick={() => handleSort('price')}
              >
                现价 <SortIcon field="price" />
              </th>
              <th
                className="px-4 py-3 text-right text-sm font-medium text-gray-300 cursor-pointer hover:text-white"
                onClick={() => handleSort('change_pct')}
              >
                涨跌幅 <SortIcon field="change_pct" />
              </th>
              <th className="px-4 py-3 text-center text-sm font-medium text-gray-300">
                预测方向
              </th>
              <th className="px-4 py-3 text-right text-sm font-medium text-gray-300">
                预测分数
              </th>
              <th className="px-4 py-3 text-center text-sm font-medium text-gray-300">
                操作
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-700">
            {paginatedStocks.map(stock => {
              const score = getScore(stock.code)
              return (
                <tr
                  key={stock.code}
                  className="hover:bg-slate-700/50 cursor-pointer"
                  onClick={() => onSelectStock(stock)}
                >
                  <td className="px-4 py-3 text-white font-mono">{stock.code}</td>
                  <td className="px-4 py-3 text-white">{stock.name}</td>
                  <td className="px-4 py-3 text-right font-mono">
                    {stock.price > 0 ? stock.price.toFixed(2) : '-'}
                  </td>
                  <td className={`px-4 py-3 text-right font-mono ${
                    stock.change_pct > 0 ? 'text-red-400' : stock.change_pct < 0 ? 'text-green-400' : 'text-gray-400'
                  }`}>
                    {stock.change_pct !== 0 ? `${stock.change_pct > 0 ? '+' : ''}${stock.change_pct.toFixed(2)}%` : '-'}
                  </td>
                  <td className="px-4 py-3 text-center">
                    {score ? (
                      <span className={`inline-block px-2 py-1 rounded text-sm ${
                        score.direction === 'up' ? 'bg-red-900/50 text-red-400' :
                        score.direction === 'down' ? 'bg-green-900/50 text-green-400' :
                        'bg-gray-700 text-gray-400'
                      }`}>
                        {score.direction === 'up' ? '↑ 涨' : score.direction === 'down' ? '↓ 跌' : '→ 平'}
                      </span>
                    ) : (
                      <span className="text-gray-500 text-sm">-</span>
                    )}
                  </td>
                  <td className={`px-4 py-3 text-right font-mono ${getScoreColor(score?.score)}`}>
                    {score ? `${(score.score * 100).toFixed(0)}%` : '-'}
                  </td>
                  <td className="px-4 py-3 text-center">
                    <button
                      className="px-3 py-1 bg-blue-600 rounded text-sm hover:bg-blue-700"
                      onClick={e => { e.stopPropagation(); onSelectStock(stock) }}
                    >
                      详情
                    </button>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="mt-4 flex items-center justify-center gap-2">
          <button
            onClick={() => setPage(p => Math.max(1, p - 1))}
            disabled={page === 1}
            className="px-3 py-1 bg-slate-700 rounded disabled:opacity-50 hover:bg-slate-600"
          >
            上一页
          </button>
          <span className="text-gray-400 text-sm">
            第 {page} / {totalPages} 页
          </span>
          <button
            onClick={() => setPage(p => Math.min(totalPages, p + 1))}
            disabled={page === totalPages}
            className="px-3 py-1 bg-slate-700 rounded disabled:opacity-50 hover:bg-slate-600"
          >
            下一页
          </button>
        </div>
      )}
    </div>
  )
}

export default StockList
