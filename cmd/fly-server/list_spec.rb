require 'time'
require 'securerandom'

RSpec.describe 'LIST' do
    ['admin', 'regular user'].each do |persona|
        context "as #{persona}" do
            before(:all) do
                @session = as(persona)
                @session.cmd!('MKDIR', 'list-admin')

                @files = [
                    {path: 'list-admin/file1.txt', content: "hello\nworld\n"},
                    {path: 'list-admin/file2.txt', content: "hello\nworld\nfoo\nbar\n"},
                    {path: 'list-admin/file3.txt', content: "hello\nworld\nfoo\nbar\nsomething\nelse\n"}
                ]

                @files.each do |f| 
                    @session.write_file(f[:path], f[:content])
                    f[:mtime] = Time.now
                end

                @session.cmd!('MKDIR', 'list-admin/folderthing')
            end

            describe 'folder' do
                it 'returns list of files' do
                    resp = @session.cmd('LIST', 'list-admin')
                    expect(resp).to be_a(Wire::Table)
                    expect(resp.row_count).to eq(4)
                    expect(resp.col_count).to eq(4)

                    @files.each_with_index do |f, i|
                        expect(resp[i][0]).to be_a(Wire::String)
                        expect(resp[i][0].value).to eq('F')

                        expect(resp[i][1]).to be_a(Wire::String)
                        expect(resp[i][1].value).to eq(f[:path].split('/')[1])

                        expect(resp[i][2]).to be_a(Wire::Integer)
                        expect(resp[i][2].value).to eq(f[:content].length)

                        expect(resp[i][3]).to be_a(Wire::String)
                        mtime = DateTime.strptime(resp[i][3].value, '%Y-%m-%dT%H:%M:%S.%NZ')
                        expect(mtime.to_time).to be_within(0.1).of(f[:mtime])
                    end

                    expect(resp[3][0]).to be_a(Wire::String)
                    expect(resp[3][0].value).to eq('D')

                    expect(resp[3][1]).to be_a(Wire::String)
                    expect(resp[3][1].value).to eq('folderthing')

                    expect(resp[3][2]).to be_a(Wire::Null)

                    expect(resp[3][3]).to be_a(Wire::String)
                    mtime = DateTime.strptime(resp[3][3].value, '%Y-%m-%dT%H:%M:%S.%NZ')
                    expect(mtime.to_time).to be_within(0.100).of(Time.now)
                end

                it 'does not show .fly' do
                    resp = @session.cmd('LIST', '/')
                    expect(resp).to be_a(Wire::Table)

                    resp.each do |f|
                        expect(f[1].value).not_to eq('.fly')
                    end
                end
            end

            describe 'file' do
                it 'returns file stats' do
                    resp = @session.cmd('LIST', 'list-admin/file2.txt')
                    expect(resp).to be_a(Wire::Table)
                    expect(resp.row_count).to eq(1)
                    expect(resp.col_count).to eq(4)

                    expect(resp[0][0]).to be_a(Wire::String)
                    expect(resp[0][0].value).to eq('F')

                    expect(resp[0][1]).to be_a(Wire::String)
                    expect(resp[0][1].value).to eq('file2.txt')

                    expect(resp[0][2]).to be_a(Wire::Integer)
                    expect(resp[0][2].value).to eq(20)

                    expect(resp[0][3]).to be_a(Wire::String)
                    mtime = DateTime.strptime(resp[0][3].value, '%Y-%m-%dT%H:%M:%S.%NZ')
                    expect(mtime.to_time).to be_within(0.100).of(@files[1][:mtime])
                end
            end

            describe 'non existent path' do
                it 'returns NOTFOUND' do
                    resp = @session.cmd('LIST', "/some/path/#{SecureRandom.hex}")
                    expect(resp).to be_error('NOTFOUND')
                end
            end
        end
    end

    context 'unauthenticated' do
        it 'returns error' do
            admin.cmd!('MKDIR', 'list-unauth')
            admin.write_file('list-unauth/somefile.txt', "hello\nworld\n")
            resp = unauth.cmd('LIST', 'list-unauth')
            expect(resp).to be_error('DENIED')
        end
    end

    context 'unauthorized' do
        before(:all) do
            @username = Username.get_next
            admin.cmd!('ADDUSER', @username, 'password')
            admin.cmd!('MKDIR', "/home/#{@username}")
            admin.cmd!('MKDIR', "/home/other/#{@username}")
            admin.cmd!('MKDIR', "/usr/home/#{@username}")
            admin.cmd!('MKDIR', "/usr/home/other/#{@username}")

            @session = Session.new
            @session.cmd!('AUTH', 'PWD', @username, 'password')
        end

        after(:all) do
            @session.close
        end

        context 'file' do
            before(:all) do
                admin.write_file("/home/#{@username}/test.txt", "hello\nexplicit\ndeny\n")
                admin.write_file("/home/other/#{@username}/test.txt", "hello\nimplicit\ndeny\n")
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'R', [@username], ["/home/#{@username}"])
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'DENY', 'R', [@username], ["/home/#{@username}/test.txt"])
            end

            context 'implicit deny' do
                it 'returns DENIED' do
                    resp = @session.cmd('LIST', "/home/other/#{@username}/test.txt")
                    expect(resp).to be_error('DENIED')
                end
            end
            
            context 'explicit deny' do
                it 'returns DENIED' do
                    resp = @session.cmd('LIST', "/home/#{@username}/test.txt")
                    expect(resp).to be_error('DENIED')
                end
            end
        end

        context 'folder' do
            before(:all) do
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'R', [@username], ["/usr/home"])
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'DENY', 'R', [@username], ["/usr/home/#{@username}"])

                admin.cmd!('MKDIR', "/docs/#{@username}/homework")
                admin.write_file("/docs/#{@username}/readme.txt", "hello\nlist\n")
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'R', [@username], ["/docs/#{@username}"])
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'DENY', 'R', [@username], ["/docs/#{@username}/homework"])
            end

            context 'implicit deny' do
                it 'returns DENIED' do
                    resp = @session.cmd('LIST', "/etc/#{@username}")
                    expect(resp).to be_error('DENIED')
                end
            end

            context 'explicit deny' do
                it 'returns DENIED' do
                    resp = @session.cmd('LIST', "/usr/home/#{@username}")
                    expect(resp).to be_error('DENIED')
                end
            end

            it 'hides files without access' do
                resp = @session.cmd('LIST', "/docs/#{@username}")
                expect(resp).to be_a(Wire::Table)

                expect(resp.row_count).to eq(1)
                expect(resp.col_count).to eq(4)

                expect(resp[0][0]).to be_a(Wire::String)
                expect(resp[0][0].value).to eq('F')

                expect(resp[0][1]).to be_a(Wire::String)
                expect(resp[0][1].value).to eq('readme.txt')
            end
        end
    end
end