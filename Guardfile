# A sample Guardfile
# More info at https://github.com/guard/guard#readme    

def squash(m)
    if m[1].end_with? 'squashed'
        return
    end
    puts 'squashing js and css...'
    puts `./http_root/squash`
end

def test(m)
    dir = File.dirname(m[1])
    puts `cd #{dir} && go test`
end

guard :shell do
  watch(%r{(http_root/.+)\.(css|js)}) { |m| squash(m) }
end

guard :shell do
    watch(%r{(src/.+)\.go}) { |m| test(m) }
end